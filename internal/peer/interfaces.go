package peer

import "btc/internal/protocol"

// Connection defines the interface for peer communication.
// Allows mocking peer connections in unit tests.
type Connection interface {
	// Read reads and parses the next message from the peer
	Read() (*protocol.Message, error)

	// SendRequest sends request for a block from the peer
	SendRequest(index, begin, length int) error

	// SendInterested tells the peer we're interested in their pieces
	SendInterested() error

	// SendNotInterested tells the peer we're not interested
	SendNotInterested() error

	// SendUnchoke tells the peer we're unchoking them
	SendUnchoke() error

	// SendHave tells the peer we have a specific piece
	SendHave(index int) error

	// Close closes the connection
	Close() error

	// GetBitfield returns the peer's bitfield
	GetBitfield() protocol.Bitfield

	// IsChoked returns whether we're choked by this peer
	IsChoked() bool
}

// Ensure Client implements Connection interface
var _ Connection = (*Client)(nil)
