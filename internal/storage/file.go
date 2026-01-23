package storage

import "os"

type FileStorage struct {
	file *os.File
	piecelen int
	totalLen int
}


