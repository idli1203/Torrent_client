package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageID uint8

const (
	MsgChoke        MessageID = 0
	MsgUnchoke      MessageID = 1
	MsgInterested   MessageID = 2
	MsgUnInterested MessageID = 3
	MsgHave         MessageID = 4
	MsgBitfield     MessageID = 5
	MsgRequest      MessageID = 6
	MsgPiece        MessageID = 7
	MsgCancel       MessageID = 8
)

type Message struct {
	Payload []byte
	ID      MessageID
}

func FormatRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}

func FormatHave(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[:], uint32(index))
	return &Message{ID: MsgHave, Payload: payload}
}

func ParsePiece(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("msg ID is not MsgPiece, got id = %d", msg.ID)
	}

	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("payload is too short: %d", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("indexes not matching -- found: %d, expected: %d", parsedIndex, index)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("begin offset too high: %d >= %d", begin, len(buf))
	}

	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("data too long for offset: data=%d, offset=%d, bufLen=%d", len(data), begin, len(buf))
	}

	copy(buf[begin:], data)
	return len(data), nil
}

func ParseHave(msg *Message) (int, error) {
	if msg.ID != MsgHave {
		return 0, fmt.Errorf("message ID does not match MsgHave")
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("payload length is different than required")
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))
	return index, nil
}

func Read(r io.Reader) (*Message, error) {
	buffer := make([]byte, 4)

	_, err := io.ReadFull(r, buffer)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(buffer)

	if length == 0 {
		return nil, nil
	}

	msgBuf := make([]byte, length)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      MessageID(msgBuf[0]),
		Payload: msgBuf[1:],
	}, nil
}

func (msg *Message) Serialize() []byte {
	if msg == nil {
		return make([]byte, 4)
	}

	prefixLen := uint32(len(msg.Payload) + 1)
	buf := make([]byte, 4+prefixLen)

	binary.BigEndian.PutUint32(buf[0:4], prefixLen)
	buf[4] = byte(msg.ID)
	copy(buf[5:], msg.Payload)

	return buf
}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}

	switch m.ID {
	case MsgChoke:
		return "Choke"
	case MsgUnchoke:
		return "Unchoke"
	case MsgInterested:
		return "Interested"
	case MsgUnInterested:
		return "NotInterested"
	case MsgHave:
		return "Have"
	case MsgBitfield:
		return "Bitfield"
	case MsgCancel:
		return "Cancel"
	case MsgPiece:
		return "Piece"
	case MsgRequest:
		return "Request"
	default:
		return fmt.Sprintf("Unknown#%d", m.ID)
	}
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}
	return fmt.Sprintf("%s [%d]", m.name(), len(m.Payload))
}
