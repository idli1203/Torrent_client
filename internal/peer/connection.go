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
	Peer     Peer
	infohash [20]byte
	peerID   [20]byte
	Choke    bool
	cfg      *config.Config
}

func completeHandshake(conn net.Conn, infohash, peerID [20]byte, cfg *config.Config) (*protocol.Handshake, error) {
	conn.SetDeadline(time.Now().Add(cfg.HandshakeTimeout))
	defer conn.SetDeadline(time.Time{})

	req, err := protocol.NewHandshake(infohash, peerID)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := protocol.ReadHandshake(conn)
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
	conn, err := net.DialTimeout("tcp", peer.String(), cfg.TCPTimeout)
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
		Peer:     peer,
		infohash: infohash,
		peerID:   peerID,
		Choke:    true,
		cfg:      cfg,
	}, nil
}

func (c *Client) Read() (*protocol.Message, error) {
	return protocol.Read(c.Conn)
}

func (c *Client) SendRequest(index, begin, length int) error {
	req := protocol.FormatRequest(index, begin, length)
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

func (c *Client) SendHave(index int) error {
	msg := protocol.FormatHave(index)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *Client) GetBitfield() protocol.Bitfield {
	return c.Bitfield
}

func (c *Client) IsChoked() bool {
	return c.Choke
}

func (c *Client) Close() error {
	return c.Conn.Close()
}
