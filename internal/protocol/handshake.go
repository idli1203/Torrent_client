package protocol

import (
	"fmt"
	"io"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infohash [20]byte, peerID [20]byte) (*Handshake, error) {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infohash,
		PeerID:   peerID,
	}, nil
}

func (h *Handshake) Serialize() []byte {
	buffer := make([]byte, len(h.Pstr)+49)
	buffer[0] = byte(len(h.Pstr))

	point := 1
	point += copy(buffer[point:], h.Pstr)
	point += copy(buffer[point:], make([]byte, 8))
	point += copy(buffer[point:], h.InfoHash[:])
	point += copy(buffer[point:], h.PeerID[:])

	return buffer
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	lenBuf := make([]byte, 1)

	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}

	pstrLen := int(lenBuf[0])

	if pstrLen == 0 {
		return nil, fmt.Errorf("pstrlen cannot be zero")
	}

	handshakeBuf := make([]byte, 48+pstrLen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var peerID, infohash [20]byte

	copy(infohash[:], handshakeBuf[pstrLen+8:pstrLen+28])
	copy(peerID[:], handshakeBuf[pstrLen+28:])

	return &Handshake{
		Pstr:     string(handshakeBuf[:pstrLen]),
		PeerID:   peerID,
		InfoHash: infohash,
	}, nil
}
