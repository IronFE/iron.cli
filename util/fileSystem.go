package util

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func GetWorkDirFromArg(arg string) (string, error) {
	var workdir string
	if path.IsAbs(arg) {
		workdir = arg
	} else {
		currentDir, _ := os.Getwd()
		workdir = path.Join(currentDir, arg)
	}

	if _, err := os.Stat(workdir); os.IsNotExist(err) {
		return "", errors.Wrap(err, "the given directory does not exist")
	}

	return workdir, nil
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func CopyFile(source, destination string) error {

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Flush to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to get the file permission from %s: %w", source, err)
	}

	if err = os.Chmod(destination, info.Mode()); err != nil {
		return fmt.Errorf("failed to set the file permission to %s: %w", destination, err)
	}

	return nil
}

func CopyFolder(source, destination string) error {
	var err error = filepath.Walk(source, func(path string, info os.FileInfo, _ error) error {
		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), info.Mode())
		} else {
			var data, err1 = os.ReadFile(filepath.Join(source, relPath))
			if err1 != nil {
				return err1
			}
			return os.WriteFile(filepath.Join(destination, relPath), data, info.Mode())
		}
	})
	return err
}
