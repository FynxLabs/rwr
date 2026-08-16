package system

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

// defaultFileMode is the mode used for files these helpers create from scratch.
// An existing target keeps its own mode instead (see targetMode): a rewrite must
// never widen the permissions of a file that may hold credentials.
const defaultFileMode os.FileMode = 0644

// OpenFileNoFollow opens path while refusing a symlink as the final component
// on platforms that support that guarantee.
func OpenFileNoFollow(path string, flags int, mode os.FileMode) (*os.File, error) {
	return openFileNoFollow(path, flags, mode)
}

// tempFileNextTo creates a staging file in the target's own directory.
//
// It has to be that directory and not os.TempDir(): /tmp is tmpfs on most Linux
// distributions, and rename(2) between filesystems fails with EXDEV, so staging
// in /tmp broke every write to a target outside it. When the target's directory
// cannot be written (an elevated write to a root-owned directory), staging falls
// back to the system temp directory and moveIntoPlace copies instead.
func tempFileNextTo(target, pattern string) (*os.File, error) {
	if f, err := os.CreateTemp(filepath.Dir(target), pattern); err == nil {
		return f, nil
	}
	return os.CreateTemp("", pattern)
}

// moveIntoPlace puts the staged file at target with the given mode, removing the
// staged file on every path.
//
// A cross-filesystem rename fails with a *os.LinkError, so any rename failure
// falls back to copying the content; the fallback is what makes the system-temp
// staging path above work.
func moveIntoPlace(staged, target string, mode os.FileMode, elevated bool) error {
	defer func() {
		if removeErr := os.Remove(staged); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Debugf("could not remove staging file %s: %v", staged, removeErr)
		}
	}()

	if chmodErr := os.Chmod(staged, mode); chmodErr != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("error setting permissions on temporary file: %v", chmodErr)
	}

	if elevated {
		return moveFileWithElevatedPrivileges(staged, target)
	}

	renameErr := os.Rename(staged, target)
	if renameErr == nil {
		return nil
	}

	var linkErr *os.LinkError
	if !errors.As(renameErr, &linkErr) {
		return fmt.Errorf("error moving file: %v", renameErr)
	}

	if copyErr := copyFileContentMode(staged, target, mode); copyErr != nil {
		return fmt.Errorf("error moving file: %v (rename failed: %v)", copyErr, renameErr)
	}
	return nil
}

// copyFileContentMode copies source over target, creating target with mode and
// forcing mode onto a target that already existed.
func copyFileContentMode(source, target string, mode os.FileMode) error {
	sourceFile, err := os.Open(source) // #nosec G304 -- staging file created by this package
	if err != nil {
		return fmt.Errorf("error opening staged file: %v", err)
	}
	defer sourceFile.Close() //nolint:errcheck

	targetFile, err := OpenFileNoFollow(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("error creating target file: %v", err)
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		if cerr := targetFile.Close(); cerr != nil {
			log.Debugf("Closing %s after a failed copy: %v", target, cerr)
		}
		return fmt.Errorf("error copying file: %v", err)
	}
	if err := targetFile.Close(); err != nil {
		return fmt.Errorf("error closing target file: %v", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(target, mode); err != nil {
			return fmt.Errorf("error setting file permissions: %v", err)
		}
	}
	return nil
}

// targetMode returns the mode a write to filePath should end up with: the
// existing file's mode when there is one, so a rewrite cannot widen it.
//
// Only a file the invoking user owns donates its mode. In a shared directory
// like /tmp anyone can plant the target ahead of time, and inheriting a
// planted 0666 turns the freshly written file - root-owned, about to be
// executed by an install step - into one every local user can rewrite.
func targetMode(filePath string) os.FileMode {
	if info, err := os.Stat(filePath); err == nil {
		if !fileOwnedByEUID(info) {
			log.Warnf("Not inheriting permissions from %s: it is owned by another user; using %04o", filePath, defaultFileMode)
			return defaultFileMode
		}
		return info.Mode().Perm()
	}
	return defaultFileMode
}

// cleanupStaged removes a staging file on an error path.
// cleanupStaged removes a staging file on an error path. It is best-effort by
// design: the caller is already returning an error, and a failure to tidy up must
// not replace it - but it is logged rather than discarded, because a staging file
// left behind is the kind of thing that only shows up as a full disk much later.
func cleanupStaged(f *os.File) {
	if err := f.Close(); err != nil {
		log.Debugf("Closing staging file %s: %v", f.Name(), err)
	}
	removeStaged(f.Name())
}

// removeStaged deletes a staging file, logging rather than returning a failure.
func removeStaged(name string) {
	if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
		log.Debugf("Removing staging file %s: %v", name, err)
	}
}

// ValidateDownloadURL refuses URLs rwr must not fetch from: everything it
// downloads (package-signing keys included) is installed with the operator's
// privileges, and a plain-http fetch lets anyone on the path substitute the
// content. https is required; http is allowed only for loopback hosts, so
// local mirrors and tests keep working.
func ValidateDownloadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid download URL %q: %v", raw, err)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" {
			return nil
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
		return fmt.Errorf("refusing to download %s over plain http: the content cannot be trusted in transit; use https", raw)
	default:
		return fmt.Errorf("refusing to download %s: unsupported scheme %q", raw, u.Scheme)
	}
}

// DownloadClient is the HTTP client behind every content download rwr performs.
// Two properties the default client lacks:
//
//   - every redirect hop is re-validated with ValidateDownloadURL. Validating
//     only the initial URL is worthless the moment the server answers with a
//     302 to plain http - the default client follows it silently and the
//     content is once again substitutable in transit.
//   - a timeout, so a stalled mirror fails the step instead of hanging the
//     whole run forever. Generous, because font archives run to hundreds of MB
//     on slow links; it exists to bound a hang, not to race the download.
var DownloadClient = NewHTTPClient(15 * time.Minute)

// NewHTTPClient returns a client that re-validates every redirect hop with
// ValidateDownloadURL, with a caller-chosen timeout.
//
// The redirect policy is the point. Validating only the URL a caller passes in
// is worthless the moment the server answers with a 302 to plain http: the
// default client follows it silently, and whatever was protected by TLS is not
// any more.
//
// That matters beyond downloads, which is why this is separate from
// DownloadClient. Go strips Authorization when a redirect crosses to a
// different host, but not when it merely drops https for http on the same one -
// so a bare client following such a hop puts the GitHub token rwr sends to
// api.github.com on the wire in cleartext. The timeout is a parameter because
// a font archive and an API call want very different ones; the redirect policy
// is not, because nothing wants a worse one.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("%w: stopped after 10", ErrTooManyRedirects)
			}
			if err := refuseSchemeDowngrade(req, via); err != nil {
				return err
			}
			return ValidateDownloadURL(req.URL.String())
		},
	}
}

// ErrInsecureRedirect reports a redirect that would leave TLS behind.
var ErrInsecureRedirect = errors.New("refusing to follow an insecure redirect")

// ErrTooManyRedirects reports a redirect chain that never terminates.
var ErrTooManyRedirects = errors.New("too many redirects")

// refuseSchemeDowngrade rejects a hop to http once any earlier hop was https.
//
// It has to run before ValidateDownloadURL, which permits http for loopback so
// that local mirrors and tests keep working. That exemption is about the URL an
// operator supplies deliberately; as a *redirect target* it is a hole, because
// a remote server chooses it. An https response redirecting to
// http://127.0.0.1 would otherwise be followed, and any local process can bind
// a loopback port and read what arrives - including the device code rwr posts
// to exchange for a GitHub token.
//
// The whole chain is checked rather than only the previous hop, so
// https -> http is refused however many times it bounces on the way.
func refuseSchemeDowngrade(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "http" {
		return nil
	}
	for _, previous := range via {
		if previous.URL.Scheme == "https" {
			return fmt.Errorf("%w: %s redirected to %s, which is not encrypted",
				ErrInsecureRedirect, previous.URL.Redacted(), req.URL.Redacted())
		}
	}
	return nil
}

// HashFileSHA256 returns the hex sha256 of a file's content.
func HashFileSHA256(path string) (string, error) {
	f, err := os.Open(path) // #nosec G304 -- callers hash paths they already manage
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyFileSHA256 compares the file's digest against the expected hex string.
func verifyFileSHA256(path, wantHex string) error {
	got, err := HashFileSHA256(path)
	if err != nil {
		return fmt.Errorf("error hashing downloaded file: %v", err)
	}
	if !strings.EqualFold(got, strings.TrimSpace(wantHex)) {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", wantHex, got)
	}
	return nil
}

func downloadFileContent(url, filePath string) error {
	if err := ValidateDownloadURL(url); err != nil {
		return err
	}

	// Send an HTTP GET request to the URL
	response, err := DownloadClient.Get(url) // #nosec G107 -- scheme restricted by ValidateDownloadURL above and per redirect hop by the client
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close() //nolint:errcheck
		if err != nil {
			log.Errorf("error closing response body: %v", err)
		}
	}(response.Body)

	// Check the response status code
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file: HTTP status %d", response.StatusCode)
	}

	// Create the file
	file, err := OpenFileNoFollow(filePath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close() //nolint:errcheck
		if err != nil {
			log.Errorf("error closing file: %v", err)
		}
	}(file)

	// Copy the response body to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func moveFileWithElevatedPrivileges(source, target string) error {
	// "--" so a path starting with "-" can never be read as an mv option:
	// these paths come from blueprint values, and the command runs as root.
	cmd := types.Command{
		Exec:     "mv",
		Args:     []string{"--", source, target},
		Elevated: true,
	}
	err := RunCommand(cmd, false)
	if err != nil {
		return fmt.Errorf("error moving file with elevated privileges: %v", err)
	}
	return nil
}

// DownloadFile downloads a file from the given URL to filePath.
// If elevated is true, the file is moved to the target using sudo.
func DownloadFile(url, filePath string, elevated bool) error {
	return DownloadFileWithChecksum(url, filePath, elevated, "")
}

// DownloadFileWithChecksum is DownloadFile with an optional expected sha256
// hex digest: when non-empty, the download is verified before it is moved
// into place and discarded on a mismatch.
func DownloadFileWithChecksum(url, filePath string, elevated bool, sha256Hex string) error {

	log.Debugf("Downloading file from %s to %s", url, filePath)
	mode := targetMode(filePath)

	tempFile, err := tempFileNextTo(filePath, "download-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		removeStaged(tempFile.Name())
		return fmt.Errorf("error closing temporary file: %v", err)
	}

	if err := downloadFileContent(url, tempFile.Name()); err != nil {
		removeStaged(tempFile.Name())
		return err
	}

	if sha256Hex != "" {
		if err := verifyFileSHA256(tempFile.Name(), sha256Hex); err != nil {
			removeStaged(tempFile.Name())
			return fmt.Errorf("refusing to install %s: %w", url, err)
		}
	}

	return moveIntoPlace(tempFile.Name(), filePath, mode, elevated)
}

// WriteToFile writes content to a file, replacing any existing content.
// If elevated is true, the write is performed with sudo privileges.
func WriteToFile(filePath, content string, elevated bool) error {

	log.Debugf("Writing content to file %s", filePath)
	return replaceFileContent(filePath, content, elevated)
}

// AppendToFile adds content to the end of a file, creating it when absent.
//
// Unlike WriteToFile this keeps what is already there: the pacman family adds a
// repository by appending a section to /etc/pacman.conf, and a truncating write
// destroys every other repository and option in it.
//
// The append is idempotent because `rwr all` is expected to be run repeatedly:
// content already present in the file is left alone rather than added a second
// time.
func AppendToFile(filePath, content string, elevated bool) error {
	log.Debugf("Appending content to file %s", filePath)

	existing, err := os.ReadFile(filePath) // #nosec G304 -- path is provider-declared and contained by the caller
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error reading %s: %v", filePath, err)
	}

	block := strings.TrimSpace(content)
	if block == "" {
		return nil
	}
	if containsBlock(string(existing), block) {
		log.Debugf("Content already present in %s, not appending again", filePath)
		return nil
	}

	var next strings.Builder
	next.Write(existing)
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		next.WriteString("\n")
	}
	next.WriteString(block)
	next.WriteString("\n")

	return replaceFileContent(filePath, next.String(), elevated)
}

// RemoveLineFromFile deletes every line of a file that names match, leaving the
// file alone when it holds no such line and when it does not exist at all.
//
// A line names match when its whole content is match or when one of its
// whitespace-separated fields is: a mirrors file line is often the URL followed
// by options, and a substring test would also delete an unrelated longer URL.
func RemoveLineFromFile(filePath, match string, elevated bool) error {
	log.Debugf("Removing lines naming %s from file %s", match, filePath)

	if strings.TrimSpace(match) == "" {
		return fmt.Errorf("remove_line step has no match")
	}

	existing, err := os.ReadFile(filePath) // #nosec G304 -- path is provider-declared and contained by the caller
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("File already absent: %s", filePath)
			return nil
		}
		return fmt.Errorf("error reading %s: %v", filePath, err)
	}

	kept := make([]string, 0)
	removed := false
	for _, line := range strings.Split(string(existing), "\n") {
		if lineNames(line, strings.TrimSpace(match)) {
			removed = true
			continue
		}
		kept = append(kept, line)
	}

	if !removed {
		return nil
	}

	return replaceFileContent(filePath, strings.Join(kept, "\n"), elevated)
}

// RemoveSectionFromFile deletes an ini-style "[section]" header and the lines
// that follow it, up to the next header or the end of the file. A file without
// that section, or no file at all, is already in the requested state.
func RemoveSectionFromFile(filePath, section string, elevated bool) error {
	log.Debugf("Removing section %s from file %s", section, filePath)

	name := strings.TrimSpace(section)
	if name == "" {
		return fmt.Errorf("remove_section step has no section")
	}
	header := "[" + name + "]"

	existing, err := os.ReadFile(filePath) // #nosec G304 -- path is provider-declared and contained by the caller
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("File already absent: %s", filePath)
			return nil
		}
		return fmt.Errorf("error reading %s: %v", filePath, err)
	}

	kept := make([]string, 0)
	inSection := false
	removed := false
	for _, line := range strings.Split(string(existing), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = trimmed == header
			if inSection {
				removed = true
				continue
			}
		}
		if inSection {
			continue
		}
		kept = append(kept, line)
	}

	if !removed {
		return nil
	}

	return replaceFileContent(filePath, strings.Join(kept, "\n"), elevated)
}

// replaceFileContent stages content next to the target and moves it into place,
// keeping the target's existing mode.
func replaceFileContent(filePath, content string, elevated bool) error {
	mode := targetMode(filePath)

	tempFile, err := tempFileNextTo(filePath, "temp-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}

	if _, err := tempFile.WriteString(content); err != nil {
		cleanupStaged(tempFile)
		return fmt.Errorf("error writing to temporary file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		removeStaged(tempFile.Name())
		return fmt.Errorf("error closing temporary file: %v", err)
	}

	return moveIntoPlace(tempFile.Name(), filePath, mode, elevated)
}

// containsBlock reports whether every line of block already appears in file, in
// order and consecutively. Lines are compared with surrounding whitespace
// removed so that re-indented or differently terminated content still counts as
// present.
func containsBlock(file, block string) bool {
	fileLines := trimmedLines(file)
	blockLines := trimmedLines(block)
	if len(blockLines) == 0 || len(blockLines) > len(fileLines) {
		return false
	}

	for start := 0; start+len(blockLines) <= len(fileLines); start++ {
		match := true
		for i, want := range blockLines {
			if fileLines[start+i] != want {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func trimmedLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func lineNames(line, match string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if trimmed == match {
		return true
	}
	for _, field := range strings.Fields(trimmed) {
		if field == match {
			return true
		}
	}
	return false
}

// CopyFile copies a file from source to target, preserving permissions.
// If elevated is true, the copy is performed with sudo privileges.
func CopyFile(source, target string, elevated bool, osInfo *types.OSInfo) error {
	log.Debugf("Copying file from %s to %s (elevated: %v)", source, target, elevated)

	sourceFile, err := os.Open(source) // #nosec G304 -- path is operator-supplied blueprint/config input
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer sourceFile.Close() //nolint:errcheck

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("error getting source file info: %v", err)
	}

	// The directory has to be created with the same privilege as the copy:
	// an elevated copy into /etc/new-dir ran os.MkdirAll unelevated first and
	// died with EACCES before the elevated mv ever ran.
	targetDir := filepath.Dir(target)
	if elevated {
		mkdir := types.Command{Exec: "mkdir", Args: []string{"-p", "--", targetDir}, Elevated: true}
		if err := RunCommand(mkdir, false); err != nil {
			return fmt.Errorf("error creating target directory with elevated privileges: %v", err)
		}
		// 0755, not os.ModePerm: 0777 through a permissive umask is a directory
		// every local user can write, on a path that may hold installed content.
	} else if err := os.MkdirAll(targetDir, 0o755); err != nil { // #nosec G301 -- parent of a blueprint-named target; 0755 so installed content stays world-readable, never world-writable
		return fmt.Errorf("error creating target directory: %v", err)
	}

	if elevated {
		tempFile, err := os.CreateTemp("", "rwr-copy-")
		if err != nil {
			return fmt.Errorf("error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name()) //nolint:errcheck

		_, err = io.Copy(tempFile, sourceFile)
		if err != nil {
			return fmt.Errorf("error copying to temporary file: %v", err)
		}

		err = tempFile.Close() //nolint:errcheck
		if err != nil {
			return fmt.Errorf("error closing temporary file: %v", err)
		}

		err = moveFileWithElevatedPrivileges(tempFile.Name(), target)
		if err != nil {
			return fmt.Errorf("error moving file with elevated privileges: %v", err)
		}
	} else {
		targetFile, err := OpenFileNoFollow(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
		if err != nil {
			return fmt.Errorf("error creating target file: %v", err)
		}
		defer targetFile.Close() //nolint:errcheck

		_, err = io.Copy(targetFile, sourceFile)
		if err != nil {
			return fmt.Errorf("error copying file: %v", err)
		}
		if runtime.GOOS != "windows" {
			if err := targetFile.Chmod(sourceInfo.Mode()); err != nil {
				return fmt.Errorf("error setting file permissions: %v", err)
			}
		}
	}

	// Elevated writes no longer have an open destination descriptor, so their
	// permission update remains a separate privileged operation.
	if runtime.GOOS != "windows" && elevated {
		err = setFilePermissionsElevated(target, sourceInfo.Mode())
		if err != nil {
			return fmt.Errorf("error setting file permissions: %v", err)
		}
	}

	return nil
}

func setFilePermissionsElevated(path string, mode os.FileMode) error {
	// "--" so a path starting with "-" can never be read as a chmod option:
	// the path comes from blueprint values, and the command runs as root.
	cmd := types.Command{
		Exec:     "chmod",
		Args:     []string{fmt.Sprintf("%o", mode), "--", path},
		Elevated: true,
	}
	return RunCommand(cmd, false)
}

// ExpandPath replaces a leading "~" or "~/" in the path with the user's home
// directory. A bare "~" has to be handled too, or a blueprint target of "~"
// creates a directory literally named "~".
func ExpandPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// CopyDirectory recursively copies a directory from source to target.
// In interactive mode, it shows diffs and prompts before overwriting existing files.
func CopyDirectory(source, target string, elevated, interactive bool) error {
	log.Debugf("Copying directory from %s to %s", source, target)

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(target, relPath)

		if info.IsDir() {
			// Created with the same privilege as the copy: `elevated` used to be
			// accepted and ignored, so a directories blueprint with elevated: true
			// failed with EACCES on the first root-owned target.
			if elevated {
				mkdir := types.Command{Exec: "mkdir", Args: []string{"-p", "--", targetPath}, Elevated: true}
				if err := RunCommand(mkdir, false); err != nil {
					return fmt.Errorf("error creating directory %s with elevated privileges: %v", targetPath, err)
				}
			} else if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return err
			}
		} else {
			// Check if the file already exists at the target path
			_, err := os.Stat(targetPath)
			if err == nil {
				// File exists, prompt for confirmation if interactive mode is
				// enabled. The whole interaction - notice, diff, question -
				// holds the terminal lease: printing a multi-line diff and
				// reading stdin over the dashboard tore the viewport and
				// fought it for keystrokes.
				if interactive {
					var overwrite bool
					promptErr := reporting.WithTerminal(func() error {
						fmt.Printf("File '%s' already exists at the target location.\n", targetPath)
						fmt.Printf("Diff:\n")
						if diffErr := ShowDiff(path, targetPath); diffErr != nil {
							log.Errorf("Failed to show diff: %v", diffErr)
						}
						var innerErr error
						overwrite, innerErr = promptOverwrite()
						return innerErr
					})
					if promptErr != nil {
						return promptErr
					}
					if !overwrite {
						log.Infof("Skipping file: %s", targetPath)
						return nil
					}
				}

				// Overwrite the file. The elevated path replaces it via an
				// elevated mv inside CopyFile, so only the unelevated copy has
				// to clear the target itself.
				if !elevated {
					if err := os.Remove(targetPath); err != nil {
						return err
					}
				}
			}

			// Copy the file with the privilege the blueprint asked for.
			if err := CopyFile(path, targetPath, elevated, nil); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error copying directory: %v", err)
	}

	return nil
}

// FileExists reports whether the file at filePath exists.
//
// Only a successful stat counts. Reporting "exists" for any non-ENOENT error
// meant a permission-denied parent directory looked like a hit, which is enough
// to send the bootstrap path in processors down the wrong branch.
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// LookupUID returns the numeric user ID for the given username.
func LookupUID(owner string) (int, error) {
	u, err := user.Lookup(owner)
	if err != nil {
		return -1, err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

// LookupGID returns the numeric group ID for the given group name.
func LookupGID(group string) (int, error) {
	g, err := user.LookupGroup(group)
	if err != nil {
		return -1, err
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return -1, err
	}
	return gid, nil
}

// promptOverwrite returns an error instead of dying on a failed read:
// log.Fatalf here bypassed the failure ledger and every deferred cleanup, and
// interactive defaults to true, so a piped stdin hitting EOF killed the whole
// run mid-flight.
func promptOverwrite() (bool, error) {
	var input string
	fmt.Print("Do you want to overwrite the file? (y/n): ")
	if _, err := fmt.Scanln(&input); err != nil {
		return false, fmt.Errorf("reading overwrite confirmation (run with --interactive=false to skip prompts): %w", err)
	}
	return strings.EqualFold(input, "y") || strings.EqualFold(input, "yes"), nil
}
