package peer
import (
	"btc/internal/protocol"
	"bytes"
	"fmt"
	"net"
	"time"
)

type Client struct {
	Conn     net.Conn
	Bitfield protocol.Bitfield
	Peers    Peer
	infohash [20]byte
	peerid   [20]byte
	Choke    bool
}

// complete the handshake to transfer the files between peers by establishing a connection
func CompleteHandshake(conn net.Conn, infohash, peerid [20]byte) (*protocol.Handshake, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second)) // basically for establishing a conn within 5 seconds other wise go to other peers

	defer conn.SetDeadline(time.Time{}) // disables the deadline

	hand_req, err := protocol.NewHandshake(infohash, peerid)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(hand_req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := protocol.Read_Handshake(conn)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(res.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("The infohashes do not match , required : %v  \n got : %v", res.InfoHash[:], infohash[:])
	}

	return res, nil
}

func RecieveBitfield(conn net.Conn) (protocol.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	msg, err := protocol.Read(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		err := fmt.Errorf("Got something other than bitfield : %s", msg)
		return nil, err
	}
	if msg.ID != protocol.MsgBitfield {
		err := fmt.Errorf("Expected bitfield but got ID %d", msg.ID)
		return nil, err
	}

	return msg.Payload, nil
}

func New(peer Peer, peerID, infohash [20]byte) (* Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = CompleteHandshake(conn, infohash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bfield, err := RecieveBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn:     conn,
		Bitfield: bfield,
		Peers:    peer,
		infohash: infohash,
		peerid:   peerID,
		Choke:    true,
	}, nil
}

func (c *Client) Read() (*protocol.Message, error) {
	msg, err := protocol.Read(c.Conn)
	return msg, err
}

// Sends a request message to the peer
func (c *Client) SendRequest(idx, begin, len int) error {
	req := protocol.Format_msgRequest(idx, begin, len)
	_, err := c.Conn.Write(req.Serialize())

	return err
}

func (c *Client) SendInterested() error {
	msg := protocol.Message{ID: protocol.MsgInterested}

	_, err := c.Conn.Write(msg.Serialize())

	return err
}

func (c *Client) SendNotInterested() error {
	msg := protocol.Message{ID: protocol.MsgUnInterested}

	_, err := c.Conn.Write(msg.Serialize())

	return err
}

func (c *Client) SendUnchoke() error {
	msg := protocol.Message{ID: protocol.MsgUnchoke}

	_, err := c.Conn.Write(msg.Serialize())

	return err
}

func (c *Client) SendHave(idx int) error {
	msg := protocol.Format_msgHave(idx)

	_, err := c.Conn.Write(msg.Serialize())

	return err
}
