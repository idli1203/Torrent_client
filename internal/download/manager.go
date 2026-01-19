package download

import (
	"btc/internal/config"
	"btc/internal/peer"
	"btc/internal/protocol"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"runtime"
	"time"
)

type Torrent struct {
	PieceHashes [][20]byte
	Name        string
	Peers       []peer.Peer
	Length      int
	PieceLength int
	PeerID      [20]byte
	InfoHash    [20]byte
	Cfg         *config.Config // Config stored in struct
}

type curr_piece struct {
	index  int
	hash   [20]byte
	length int
}

type piece_res struct {
	buffer []byte
	index  int
}

type piece_progress struct {
	client     *peer.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
	index      int
}

func (status *piece_progress) read_message() error {
	msg, err := status.client.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
	}

	switch msg.ID {
	case protocol.MsgUnchoke:
		status.client.Choke = false
	case protocol.MsgChoke:
		status.client.Choke = true
	case protocol.MsgHave:
		index, err := protocol.ParseHave(msg)
		if err != nil {
			return err
		}
		status.client.Bitfield.SetPiece(index)
	case protocol.MsgPiece:
		n, err := protocol.ParsePiece(status.index, status.buf, msg)
		if err != nil {
			return err
		}
		status.downloaded += n
		status.backlog--
	}

	return nil
}

func (t *Torrent) downloadPiece(c *peer.Client, cp *curr_piece) ([]byte, error) {
	status := piece_progress{
		index:  cp.index,
		client: c,
		buf:    make([]byte, cp.length),
	}

	// Use config value for piece timeout
	c.Conn.SetDeadline(time.Now().Add(t.Cfg.PieceTimeout))
	defer c.Conn.SetDeadline(time.Time{})

	for status.downloaded < cp.length {

		if !status.client.Choke {
			// Use config values for backlog and block size
			for status.backlog < t.Cfg.RequestBacklog && status.requested < cp.length {

				blockSize := t.Cfg.BlockSize

				if cp.length-status.requested < blockSize {
					blockSize = cp.length - status.requested
				}

				err := c.SendRequest(cp.index, status.requested, blockSize)
				if err != nil {
					return nil, err
				}

				status.backlog++
				status.requested += blockSize
			}
		}

		err := status.read_message()
		if err != nil {
			return nil, err
		}
	}

	return status.buf, nil
}

func integrityCheck(cp *curr_piece, buf []byte) error {
	hashed := sha1.Sum(buf)

	if !bytes.Equal(hashed[:], cp.hash[:]) {
		return fmt.Errorf("%d failed integrity check", cp.index)
	}

	return nil
}

func (t *Torrent) startDownload(p peer.Peer, workQueue chan *curr_piece, results chan *piece_res) {
	c, err := peer.New(p, t.PeerID, t.InfoHash, t.Cfg)
	if err != nil {
		log.Printf("Could not handshake with %s.", p.IP)
		return
	}
	defer c.Conn.Close()

	log.Printf("Successful handshake with %s\n", p.IP)

	c.SendUnchoke()
	c.SendInterested()

	for cp := range workQueue {
		if !c.Bitfield.HasPiece(cp.index) {
			workQueue <- cp
			continue
		}

		buf, err := t.downloadPiece(c, cp)
		if err != nil {
			log.Println("Exit", err)
			workQueue <- cp
			return
		}

		err = integrityCheck(cp, buf)
		if err != nil {
			log.Printf("Piece #%d failed integrity check\n", cp.index)
			workQueue <- cp
			continue
		}

		c.SendHave(cp.index)
		results <- &piece_res{buf, cp.index}
	}
}

func (t *Torrent) BoundsForPiece(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

func (t *Torrent) PieceSize(index int) int {
	begin, end := t.BoundsForPiece(index)
	return end - begin
}

// Download stores the entire file in memory.
func (t *Torrent) Download() ([]byte, error) {
	log.Println("Starting download for", t.Name)

	workQueue := make(chan *curr_piece, len(t.PieceHashes))
	results := make(chan *piece_res)
	for index, hash := range t.PieceHashes {
		length := t.PieceSize(index)
		workQueue <- &curr_piece{index, hash, length}
	}

	for _, p := range t.Peers {
		go t.startDownload(p, workQueue, results)
	}

	// Collect results into a buffer until full
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-results
		begin, end := t.BoundsForPiece(res.index)
		copy(buf[begin:end], res.buffer)
		donePieces++

		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		numWorkers := runtime.NumGoroutine() - 1
		log.Printf("(%0.2f%%) Downloaded piece #%d from %d peers\n", percent, res.index, numWorkers)
	}

	close(workQueue)

	return buf, nil
}
