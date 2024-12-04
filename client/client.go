package client

import (
	"bitorrent_client/bitfield"
	"bitorrent_client/message"
	"bitorrent_client/peers"
	handshake "bitorrent_client/tcp_conn"
	"bytes"
	"fmt"
	"net"
	"time"
)

type Client struct {
	Conn     net.Conn
	bitfield bitfield.Bitfield
	Peers    peers.Peer
	infohash [20]byte
	peerid   [20]byte
	Choke    bool
}

// complete the handshake to transfer the files between peers by establishing a connection
func completeHandshake(conn net.Conn, infohash, peerid [20]byte) (*handshake.Handshake, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second)) // basically for establishing a conn within 5 seconds other wise go to other peers

	defer conn.SetDeadline(time.Time{}) // disables the deadline

	hand_req := handshake.Newhandshake(infohash, peerid)
	_, err := conn.Write(hand_req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := handshake.Read_Handshake(conn)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(res.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("The infohashes do not match , required : %v  \n got : %v", res.InfoHash[:], infohash[:])
	}

	return res, nil
}

func recievebitField(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		err := fmt.Errorf("Got something other than bitfield : %s", msg)
		return nil, err
	}
	if msg.ID != message.MsgBitfield {
		err := fmt.Errorf("Expected bitfield but got ID %d", msg.ID)
		return nil, err
	}

	return msg.Payload, nil
}

func New(peer peers.Peer, peerID, infohash [20]byte) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, infohash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bfield, err := recievebitField(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn:     conn,
		bitfield: bfield,
		Peers:    peer,
		infohash: infohash,
		peerid:   peerID,
		Choke:    true,
	}, nil
}

func (c *Client) Read() (*message.Message, error) {
	msg, err := message.Read(c.Conn)
	return msg, err
}

// Sends a request message to the peer
func (c *Client) SendRequest(idx, begin, len int) error {
	req := message.Format_msgRequest(idx, begin, len)
	_, err := c.Conn.Write(req.Serialize())

	return err
}

func (c *Client) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}

	_, err := c.Conn.Write(msg.Serialize())

	return err
}

func (c *Client) SendNotInterested() error {
	msg := message.Message{ID: message.MsgUnInterested}

	_, err := c.Conn.Write(msg.Serialize())

	return err
}

func (c *Client) SendUnchoke() error {
	msg := message.Message{ID: message.MsgUnchoke}

	_, err := c.Conn.Write(msg.Serialize())

	return err
}

func (c *Client) SendHave(idx int) error {
	msg := message.Format_msgHave(idx)

	_, err := c.Conn.Write(msg.Serialize())

	return err
}
