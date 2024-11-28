package helpers

import (
	"io"
	"os"
)

func ReadFileToString(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}
