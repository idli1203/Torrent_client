package storage

// Storage defines the interface for piece storage.
// This allows mocking disk I/O in tests and future implementations
// like memory-only storage or distributed storage.
type Storage interface {
	// WritePiece writes a completed piece to storage
	WritePiece(index int, data []byte) error

	// ReadPiece reads a piece from storage
	ReadPiece(index int) ([]byte, error)

	// HasPiece checks if a piece exists in storage
	HasPiece(index int) bool

	// Preallocate reserves disk space for the download
	Preallocate(totalSize int64) error

	// Close closes the storage and flushes any pending writes
	Close() error
}

// ResumeState defines the interface for download state persistence.
// This enables resuming interrupted downloads.
type ResumeState interface {
	// Save persists the current download state
	Save(completedPieces []bool, downloaded int64) error

	// Load retrieves the saved download state
	Load() (completedPieces []bool, downloaded int64, err error)

	// Exists checks if a resume state exists
	Exists() bool

	// Delete removes the resume state file
	Delete() error
}
