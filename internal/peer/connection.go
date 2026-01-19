package peer

import (
	"btc/internal/config"
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
	cfg      *config.Config
}

// completeHandshake transfers files between peers by establishing a connection
func completeHandshake(conn net.Conn, infohash, peerid [20]byte, cfg *config.Config) (*protocol.Handshake, error) {
	conn.SetDeadline(time.Now().Add(cfg.HandshakeTimeout))
	defer conn.SetDeadline(time.Time{})

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
		return nil, fmt.Errorf("infohashes do not match, required: %v, got: %v", res.InfoHash[:], infohash[:])
	}

	return res, nil
}

func receiveBitfield(conn net.Conn, cfg *config.Config) (protocol.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(cfg.HandshakeTimeout))
	defer conn.SetDeadline(time.Time{})

	msg, err := protocol.Read(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("got nil message instead of bitfield")
	}
	if msg.ID != protocol.MsgBitfield {
		return nil, fmt.Errorf("expected bitfield but got ID %d", msg.ID)
	}

	return msg.Payload, nil
}

// New creates a new peer client connection
func New(peer Peer, peerID, infohash [20]byte, cfg *config.Config) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), cfg.TcpTimeout)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, infohash, peerID, cfg)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bfield, err := receiveBitfield(conn, cfg)
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
		cfg:      cfg,
	}, nil
}

func (c *Client) Read() (*protocol.Message, error) {
	msg, err := protocol.Read(c.Conn)
	return msg, err
}

func (c *Client) SendRequest(idx, begin, length int) error {
	req := protocol.Format_msgRequest(idx, begin, length)
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

// GetBitfield returns the peer's bitfield
func (c *Client) GetBitfield() protocol.Bitfield {
	return c.Bitfield
}

// IsChoked returns whether we're choked by this peer
func (c *Client) IsChoked() bool {
	return c.Choke
}

// Close closes the connection to the peer
func (c *Client) Close() error {
	return c.Conn.Close()
}
