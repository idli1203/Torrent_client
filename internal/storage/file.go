package storage

import (
	"btc/internal/logger"
	"os"
)

type FileStorage struct {
	file     *os.File
	pieceLen int
	totalLen int
	bitfield []bool
}

func NewFileStorage(path string, plen int, tlen int) (*FileStorage, error) {

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logger.Error("failed to open file ", "path", path, "error", err)
		return nil, err
	}

	return &FileStorage{
		file:     file,
		pieceLen: plen,
		totalLen: tlen,
		bitfield: make([]bool, (tlen+plen-1)/plen),
	}, nil
}

func (fs *FileStorage) WritePiece(index int, buf []byte) error {

	var offset int64 = int64(index * fs.pieceLen)

	_, err := fs.file.WriteAt(buf, offset)
	if err != nil {
		return err
	}
	fs.bitfield[index] = true
	return nil
}

func (fs *FileStorage) HasPiece(index int) bool {
	// checking index validity
	if index < 0 || index >= len(fs.bitfield) {
		return false
	}
	return fs.bitfield[index]
}

func (fs *FileStorage) Close() error {
	return fs.file.Close()
}
