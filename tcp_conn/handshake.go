package handshake

import (
	"fmt"
	"io"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func Newhandshake(Infohash [20]byte, peerid [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent Protocol",
		InfoHash: Infohash,
		PeerID:   peerid,
	}
}

func (h *Handshake) Serialize() []byte {
	buffer := make([]byte, len(h.Pstr)+49)
	buffer[0] = byte(len(h.Pstr))

	point := 1
	point += copy(buffer[point:], (h.Pstr))
	point += copy(buffer[point:], make([]byte, 8))
	point += copy(buffer[point:], h.InfoHash[:])
	point += copy(buffer[point:], h.PeerID[:])

	return buffer
}

func Read_Handshake(r io.Reader) (*Handshake, error) {
	lenbuf := make([]byte, 1)

	_, err := io.ReadFull(r, lenbuf)
	if err != nil {
		return nil, err
	}

	pstrlen := int((lenbuf[0]))

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be zero")
		return nil, err
	}

	fmt.Println("Pstrlen", pstrlen)
	handshakebuff := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakebuff)
	if err != nil {
		return nil, err
	}

	var Peerid, infohash [20]byte

	copy(infohash[:], handshakebuff[pstrlen+8:pstrlen+28])
	copy(Peerid[:], handshakebuff[pstrlen+28:])

	h := Handshake{
		Pstr:     "BitTorrent Protocol",
		PeerID:   Peerid,
		InfoHash: infohash,
	}

	return &h, nil
}
