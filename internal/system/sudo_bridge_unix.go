//go:build !windows

package system

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"charm.land/log/v2"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/types"
)

const askpassInvocation = "__rwr_sudo_askpass"         // #nosec G101 -- Internal subcommand name, not a credential.
const askpassSocketEnv = "RWR_INTERNAL_ASKPASS_SOCKET" // #nosec G101 -- Environment variable name, not a credential.
const askpassTokenEnv = "RWR_INTERNAL_ASKPASS_TOKEN"   // #nosec G101 -- Environment variable name; its value is generated with crypto/rand.

// sudo executes SUDO_ASKPASS without arguments we control. A private helper
// re-enters this executable; init also works when that executable is a test binary.
// Only endpoint/capability metadata enters the environment. Password bytes travel
// over a private Unix socket and sudo's askpass stdout, never through a file/argv.
func init() {
	if len(os.Args) == 2 && os.Args[1] == askpassInvocation {
		if err := askpassClient(os.Stdout); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func askpassClient(output io.Writer) error {
	endpoint, token := os.Getenv(askpassSocketEnv), os.Getenv(askpassTokenEnv)
	if endpoint == "" || len(token) != 64 {
		return errors.New("missing askpass capability")
	}
	conn, err := net.DialTimeout("unix", endpoint, 5*time.Second) // #nosec G704 -- Network is hardcoded to unix: local IPC cannot make an HTTP/network SSRF request.
	if err != nil {
		return err
	}
	defer closeAskpassResource(conn)
	if _, err = io.WriteString(conn, token+" "+strconv.Itoa(os.Getppid())+"\n"); err != nil {
		return err
	}
	// The user may take time to answer the masked prompt; server cancellation
	// closes the connection rather than imposing a password-entry timeout.
	password, err := bufio.NewReader(io.LimitReader(conn, 65537)).ReadBytes('\n')
	defer zeroBytes(password)
	if err != nil {
		return err
	}
	_, err = output.Write(password)
	return err
}

// installSudoBridge is per child command, not per run. Its listener and all
// accepted connections are closed before the private directory is removed.
func installSudoBridge(command *exec.Cmd, spec types.Command) (func(), error) {
	if spec.Interactive || !spec.Escalates || spec.Elevated || spec.AsUser != "" || !reporting.SupportsInlinePrompts() {
		return func() {}, nil
	}
	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "rwr-askpass-")
	if err != nil {
		return nil, err
	}
	// Unix socket address limits are smaller than some user TMPDIR paths.
	if len(filepath.Join(dir, "socket")) > 100 {
		removeAskpassDirectory(dir)
		dir, err = os.MkdirTemp("/tmp", "rwr-askpass-")
		if err != nil {
			return nil, err
		}
	}
	cleanupDir := func() { removeAskpassDirectory(dir) }
	listener, err := net.Listen("unix", filepath.Join(dir, "socket"))
	if err != nil {
		cleanupDir()
		return nil, err
	}
	if err = os.Chmod(filepath.Join(dir, "socket"), 0600); err != nil {
		closeAskpassResource(listener)
		cleanupDir()
		return nil, err
	}
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		closeAskpassResource(listener)
		cleanupDir()
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)
	zeroBytes(tokenBytes)
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	helper := filepath.Join(dir, "askpass")
	// #nosec G306 -- Sudo executes this owner-only helper inside a private 0700 directory; it contains no password.
	if err = os.WriteFile(helper, []byte("#!/bin/sh\nexec "+quote(executable)+" "+askpassInvocation+"\n"), 0700); err != nil {
		closeAskpassResource(listener)
		cleanupDir()
		return nil, err
	}
	// The wrapper forces askpass even for children that try a terminal first.
	// An explicit nested sudo -n remains noninteractive, as requested by its caller.
	wrapper := "#!/bin/sh\nexec " + quote(sudoPath) + " -A \"$@\"\n"
	// #nosec G306 -- Owner-only executable wrapper in the private 0700 directory; contains no password.
	if err = os.WriteFile(filepath.Join(dir, "sudo"), []byte(wrapper), 0700); err != nil {
		closeAskpassResource(listener)
		cleanupDir()
		return nil, err
	}
	var mu sync.Mutex
	connections := map[net.Conn]bool{}
	closed := false
	requests := map[string]int{}
	ctx, cancel := context.WithCancel(RunContext())
	var handlers sync.WaitGroup
	var acceptDone = make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			conn, e := listener.Accept()
			if e != nil {
				return
			}
			mu.Lock()
			if closed {
				mu.Unlock()
				closeAskpassResource(conn)
				return
			}
			connections[conn] = true
			mu.Unlock()
			handlers.Add(1)
			go func() {
				defer handlers.Done()
				defer func() { closeAskpassResource(conn); mu.Lock(); delete(connections, conn); mu.Unlock() }()
				if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
					return
				}
				supplied, e := bufio.NewReader(io.LimitReader(conn, 128)).ReadString('\n')
				fields := strings.Fields(supplied)
				if e != nil || len(fields) != 2 {
					return
				}
				pid, parseErr := strconv.Atoi(fields[1])
				if parseErr != nil || pid <= 0 || subtle.ConstantTimeCompare([]byte(fields[0]), []byte(token)) != 1 {
					return
				}
				if err := conn.SetReadDeadline(time.Time{}); err != nil {
					return
				}
				mu.Lock()
				requests[fields[1]]++
				retry := requests[fields[1]] > 1
				mu.Unlock()
				if retry {
					clearSudoPassword()
				}
				password, e := validatedSudoPasswordContext(ctx)
				if e != nil {
					return
				}
				defer zeroBytes(password)
				if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
					return
				}
				input := passwordInput(password, "")
				defer zeroBytes(input)
				if _, err := conn.Write(input); err != nil {
					return
				}
			}()
		}
	}()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			mu.Lock()
			closed = true
			for conn := range connections {
				closeAskpassResource(conn)
			}
			mu.Unlock()
			closeAskpassResource(listener)
			<-acceptDone
			handlers.Wait()
			cleanupDir()
		})
	}
	// Explicit interactive commands keep the normal terminal handoff instead.
	// The -A wrapper covers normal PATH lookups. Detaching captured children
	// also prevents absolute /usr/bin/sudo from accessing /dev/tty: sudo falls back
	// to SUDO_ASKPASS when no terminal is available.
	if !spec.Interactive {
		if command.SysProcAttr == nil {
			command.SysProcAttr = &syscall.SysProcAttr{}
		}
		command.SysProcAttr.Setpgid = false
		command.SysProcAttr.Setsid = true
	}
	command.Env = overrideEnvironment(command.Env, map[string]string{"PATH": dir + string(os.PathListSeparator) + environmentValue(command.Env, "PATH"), "SUDO_ASKPASS": helper, askpassSocketEnv: filepath.Join(dir, "socket"), askpassTokenEnv: token})
	return stop, nil
}

func environmentValue(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- {
		if value, ok := strings.CutPrefix(env[i], key+"="); ok {
			return value
		}
	}
	return ""
}
func overrideEnvironment(env []string, values map[string]string) []string {
	result := make([]string, 0, len(env)+len(values))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if _, replace := values[key]; !replace {
			result = append(result, entry)
		}
	}
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	return result
}

func closeAskpassResource(resource io.Closer) {
	if err := resource.Close(); err != nil {
		log.Debugf("closing sudo askpass resource: %v", err)
	}
}
func removeAskpassDirectory(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Warnf("removing sudo askpass directory: %v", err)
	}
}
