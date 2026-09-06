package handlers

import (
	"fmt"
	"io"
	"os"
)

const maxLocalJSONFileBytes = 2 * 1024 * 1024

func readLocalJSONFile(fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxLocalJSONFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxLocalJSONFileBytes {
		return nil, fmt.Errorf("JSON file exceeds %d bytes", maxLocalJSONFileBytes)
	}
	return data, nil
}
