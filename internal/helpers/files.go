package helpers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

func DownloadFile(url, filePath string) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		log.Errorf("Error creating file: %v", err)
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("Error closing file: %v", err)
		}
	}(file)

	// Send an HTTP GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		log.Errorf("Error downloading file: %v", err)
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Errorf("Error closing response body: %v", err)
		}
	}(response.Body)

	// Check the response status code
	if response.StatusCode != http.StatusOK {
		log.Errorf("Error downloading file: HTTP status %d", response.StatusCode)
		return fmt.Errorf("error downloading file: HTTP status %d", response.StatusCode)
	}

	// Copy the response body to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Errorf("Error writing file: %v", err)
		return fmt.Errorf("error writing file: %v", err)
	}

	log.Infof("File downloaded successfully: %s", filePath)
	return nil
}

func AppendToFile(filePath, content string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("Error opening file: %v", err)
		return fmt.Errorf("error opening file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("Error closing file: %v", err)
		}
	}(file)

	_, err = file.WriteString(content)
	if err != nil {
		log.Errorf("Error appending to file: %v", err)
		return fmt.Errorf("error appending to file: %v", err)
	}

	log.Infof("Content appended to file: %s", filePath)
	return nil
}

func RemoveLineFromFile(filePath, lineToRemove string) error {
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("Error opening file: %v", err)
		return fmt.Errorf("error opening file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("Error closing file: %v", err)
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, lineToRemove) {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("Error reading file: %v", err)
		return fmt.Errorf("error reading file: %v", err)
	}

	// Truncate the file and rewrite the remaining lines
	file, err = os.OpenFile(filePath, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("Error opening file for writing: %v", err)
		return fmt.Errorf("error opening file for writing: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("Error closing file: %v", err)
		}
	}(file)

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := fmt.Fprintln(writer, line)
		if err != nil {
			log.Errorf("Error writing line to file: %v", err)
			return fmt.Errorf("error writing line to file: %v", err)
		}
	}
	err = writer.Flush()
	if err != nil {
		log.Errorf("Error flushing writer: %v", err)
		return fmt.Errorf("error flushing writer: %v", err)
	}

	log.Infof("Line removed from file: %s", filePath)
	return nil
}

func CopyFile(source, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer func(sourceFile *os.File) {
		err := sourceFile.Close()
		if err != nil {

		}
	}(sourceFile)

	targetFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("error creating target file: %v", err)
	}
	defer func(targetFile *os.File) {
		err := targetFile.Close()
		if err != nil {

		}
	}(targetFile)

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	return nil
}

func CopyDirectory(source, target string) error {
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
			err := CopyFile(path, targetPath)
			if err != nil {
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
