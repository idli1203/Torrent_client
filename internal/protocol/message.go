package protocol
import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageID uint8

const (
	// Fixed length and no Payload
	MsgChoke MessageID = 0
	// fixlength and no payload
	MsgUnchoke MessageID = 1
	// fixed length and no payload
	MsgInterested MessageID = 2
	// fixed length and no payload
	MsgUnInterested MessageID = 3
	// fixed length and payload is zero based indexed
	MsgHave MessageID = 4
	// variable length
	MsgBitfield MessageID = 5
	// fixed length and has payload  == index  , begin , length
	MsgRequest MessageID = 6
	// variable length and same payload === index , begin , block
	MsgPiece MessageID = 7
	// fixed length and same as request payload
	MsgCancel = 8
)

type Message struct {
	Payload []byte
	ID      MessageID
}

func Format_msgRequest(idx, begin, len int) *Message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(idx))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(len))

	return &Message{ID: MsgRequest, Payload: payload}
}

func Format_msgHave(idx int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[:], uint32(idx))

	return &Message{ID: MsgHave, Payload: payload}
}

func ParsePiece(idx int, msg_buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("Msg ID is not msgPiece , got id = %d", msg.ID)
	}

	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("Payload is too short : %d", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != idx {
		return 0, fmt.Errorf("Indexes no matching -- found : %d , expected : %d", parsedIndex, idx)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(msg_buf) {
		return 0, fmt.Errorf("Beginoffset too high. %d >= %d", begin, len(msg_buf))
	}

	data := msg.Payload[8:]
	if begin+len(data) > len(msg_buf) {
		return 0, fmt.Errorf("Data == %d , greater than for offset== %d , found length == %d", len(data), begin, len(msg_buf))
	}

	copy(msg_buf[begin:], data)

	return len(data), nil
}

func ParseHave(msg *Message) (int, error) {
	if msg.ID != MsgHave {
		return 0, fmt.Errorf("The message formats do not match")
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("The payload is different than required")
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))

	return index, nil
}

// Read parses a message from a stream.

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

	message_buffer := make([]byte, length)
	_, err = io.ReadFull(r, message_buffer)
	if err != nil {
		return nil, err
	}

	m := Message{
		ID:      MessageID(message_buffer[0]),
		Payload: message_buffer[1:],
	}

	return &m, nil
}

// Serialize the message into a buffer of format <lenght prefix><MessageID><payload>
func (msg *Message) Serialize() []byte {
	if msg == nil {
		return make([]byte, 4)
	}

	prefix_len := uint32(len(msg.Payload) + 1)
	msg_buf := make([]byte, 4+prefix_len)

	binary.BigEndian.PutUint32(msg_buf[0:4], prefix_len)
	msg_buf[4] = byte(msg.ID)

	copy(msg_buf[5:], msg.Payload)

	return msg_buf
}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}

	switch m.ID {
	case MsgChoke:
		return "choke"
	case MsgUnchoke:
		return "unchoke"
	case MsgInterested:
		return "interested"
	case MsgUnInterested:
		return "uninterested"
	case MsgHave:
		return "have"
	case MsgBitfield:
		return "Bitfield"
	case MsgCancel:
		return "Cancel"
	case MsgPiece:
		return "Piece"
	case MsgRequest:
		return "Request"

	default:
		return fmt.Sprintf("Unknown %d", m.ID)
	}
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}

	return fmt.Sprintf("%s [%d]", m.name(), len(m.Payload))
}
