package storage

import (
	"encoding/json"
	"os"
)

// ResumeData holds the state needed to resume an interrupted download
type ResumeData struct {
	InfoHash        [20]byte `json:"info_hash"`
	CompletedPieces []bool   `json:"completed_pieces"`
	DownloadedBytes int64    `json:"downloaded_bytes"`
}

// SaveResume writes resume data to the specified path
func SaveResume(path string, data *ResumeData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// LoadResume reads resume data from the specified path
func LoadResume(path string) (*ResumeData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data ResumeData
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// ResumeExists checks if a resume file exists at the given path
func ResumeExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DeleteResume removes the resume file at the given path
func DeleteResume(path string) error {
	return os.Remove(path)
}
