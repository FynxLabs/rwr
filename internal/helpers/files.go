package helpers

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/types"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func downloadFileContent(url, filePath string) error {
	// Send an HTTP GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Errorf("error closing response body: %v", err)
		}
	}(response.Body)

	// Check the response status code
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file: HTTP status %d", response.StatusCode)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
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
	cmd := types.Command{
		Exec:     "mv",
		Args:     []string{source, target},
		Elevated: true,
	}
	err := RunCommand(cmd, false)
	if err != nil {
		return fmt.Errorf("error moving file with elevated privileges: %v", err)
	}
	return nil
}

func DownloadFile(url, filePath string, elevated bool) error {
	// Create a temporary file to download the content
	tempFile, err := os.CreateTemp("", "download-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}

	// Download the file content to the temporary file
	err = downloadFileContent(url, tempFile.Name())
	if err != nil {
		return err
	}

	// Move the temporary file to the target location
	if elevated {
		err = moveFileWithElevatedPrivileges(tempFile.Name(), filePath)
	} else {
		err = os.Rename(tempFile.Name(), filePath)
	}
	if err != nil {
		return fmt.Errorf("error moving file: %v", err)
	}

	return nil
}

func AppendToFile(filePath, content string, elevated bool) error {
	// Read the existing file content
	existingContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error reading file: %v", err)
	}

	// Create a temporary file with the appended content
	tempFile, err := os.CreateTemp("", "append-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}

	// Write the existing content and the appended content to the temporary file
	_, err = tempFile.Write(existingContent)
	if err != nil {
		return fmt.Errorf("error writing existing content to temporary file: %v", err)
	}
	_, err = tempFile.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing appended content to temporary file: %v", err)
	}

	// Move the temporary file to the target location
	if elevated {
		err = moveFileWithElevatedPrivileges(tempFile.Name(), filePath)
	} else {
		err = os.Rename(tempFile.Name(), filePath)
	}
	if err != nil {
		return fmt.Errorf("error moving file: %v", err)
	}

	return nil
}

func WriteToFile(filePath, content string, elevated bool) error {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "temp-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}

	// Write the content to the temporary file
	err = os.WriteFile(tempFile.Name(), []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing to temporary file: %v", err)
	}

	// Move the temporary file to the target location
	if elevated {
		err = moveFileWithElevatedPrivileges(tempFile.Name(), filePath)
	} else {
		err = os.Rename(tempFile.Name(), filePath)
	}
	if err != nil {
		return fmt.Errorf("error moving file: %v", err)
	}

	return nil
}

func RemoveLineFromFile(filePath, lineToRemove string, elevated bool) error {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("error closing file: %v", err)
		}
	}(file)

	// Read the file contents
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, lineToRemove) {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	// Write the updated content to a temporary file
	tempFile, err := os.CreateTemp("", "remove-line-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}

	writer := bufio.NewWriter(tempFile)
	for _, line := range lines {
		_, err := fmt.Fprintln(writer, line)
		if err != nil {
			return fmt.Errorf("error writing line to temporary file: %v", err)
		}
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("error flushing writer: %v", err)
	}

	// Move the temporary file to the target location
	if elevated {
		err = moveFileWithElevatedPrivileges(tempFile.Name(), filePath)
	} else {
		err = os.Rename(tempFile.Name(), filePath)
	}
	if err != nil {
		return fmt.Errorf("error moving file: %v", err)
	}

	return nil
}

func CopyFile(source, target string, elevated bool) error {
	// Create a temporary file to copy the content
	tempFile, err := os.CreateTemp("", "copy-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}

	// Copy the source file content to the temporary file
	err = copyFileContent(source, tempFile.Name())
	if err != nil {
		return err
	}

	// Move the temporary file to the target location
	if elevated {
		err = moveFileWithElevatedPrivileges(tempFile.Name(), target)
	} else {
		err = os.Rename(tempFile.Name(), target)
	}
	if err != nil {
		return fmt.Errorf("error moving file: %v", err)
	}

	return nil
}

func copyFileContent(source, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer func(sourceFile *os.File) {
		err := sourceFile.Close()
		if err != nil {
			log.Errorf("error closing source file: %v", err)
		}
	}(sourceFile)

	targetFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("error creating target file: %v", err)
	}
	defer func(targetFile *os.File) {
		err := targetFile.Close()
		if err != nil {
			log.Errorf("error closing target file: %v", err)
		}
	}(targetFile)

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	return nil
}

func moveDirectoryWithElevatedPrivileges(source, target string) error {
	cmd := types.Command{
		Exec:     "mv",
		Args:     []string{source, target},
		Elevated: true,
	}
	err := RunCommand(cmd, false)
	if err != nil {
		return fmt.Errorf("error moving directory with elevated privileges: %v", err)
	}
	return nil
}

func copyDirectoryContent(source, target string) error {
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
			err := os.MkdirAll(targetPath, info.Mode())
			if err != nil {
				return err
			}
		} else {
			err := copyFileContent(path, targetPath)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error copying directory content: %v", err)
	}

	return nil
}

func CopyDirectory(source, target string, elevated bool) error {
	// Create a temporary directory to copy the content
	tempDir, err := os.MkdirTemp("", "copy-dir-")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %v", err)
	}

	// Copy the source directory content to the temporary directory
	err = copyDirectoryContent(source, tempDir)
	if err != nil {
		return err
	}

	// Move the temporary directory to the target location
	if elevated {
		err = moveDirectoryWithElevatedPrivileges(tempDir, target)
	} else {
		err = os.Rename(tempDir, target)
	}
	if err != nil {
		return fmt.Errorf("error moving directory: %v", err)
	}

	return nil
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
