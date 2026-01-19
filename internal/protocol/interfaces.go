package protocol

import "io"

// MessageReader defines the interface for reading protocol messages.
type MessageReader interface {
	Read(r io.Reader) (*Message, error)
}

// MessageWriter defines the interface for writing protocol messages. (aka bencoding )
type MessageWriter interface {
	Serialize() []byte
}
