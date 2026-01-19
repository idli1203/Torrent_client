package download

import (
	"btc/internal/config"
	"btc/internal/peer"
	"btc/internal/protocol"
	"bytes"
	"context"
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
	Cfg         *config.Config
}

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	buffer []byte
	index  int
}

type pieceProgress struct {
	client     *peer.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
	index      int
}

func (state *pieceProgress) readMessage() error {
	msg, err := state.client.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
	}

	switch msg.ID {
	case protocol.MsgUnchoke:
		state.client.Choke = false
	case protocol.MsgChoke:
		state.client.Choke = true
	case protocol.MsgHave:
		index, err := protocol.ParseHave(msg)
		if err != nil {
			return err
		}
		state.client.Bitfield.SetPiece(index)
	case protocol.MsgPiece:
		n, err := protocol.ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}

	return nil
}

func (t *Torrent) downloadPiece(c *peer.Client, pw *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	c.Conn.SetDeadline(time.Now().Add(t.Cfg.PieceTimeout))
	defer c.Conn.SetDeadline(time.Time{})

	for state.downloaded < pw.length {
		if !state.client.Choke {
			for state.backlog < t.Cfg.RequestBacklog && state.requested < pw.length {
				blockSize := t.Cfg.BlockSize
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				err := c.SendRequest(pw.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}

				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
}

func checkIntegrity(pw *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.hash[:]) {
		return fmt.Errorf("piece %d failed integrity check", pw.index)
	}
	return nil
}

func (t *Torrent) startWorker(ctx context.Context, p peer.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := peer.New(p, t.PeerID, t.InfoHash, t.Cfg)
	if err != nil {
		log.Printf("Could not handshake with %s: %v", p.IP, err)
		return
	}
	defer c.Close()

	log.Printf("Successful handshake with %s", p.IP)

	c.SendUnchoke()
	c.SendInterested()

	for {
		select {
		case <-ctx.Done():
			return
		case pw, ok := <-workQueue:
			if !ok {
				return
			}

			if !c.Bitfield.HasPiece(pw.index) {
				workQueue <- pw
				continue
			}

			buf, err := t.downloadPiece(c, pw)
			if err != nil {
				log.Printf("Exiting worker: %v", err)
				workQueue <- pw
				return
			}

			err = checkIntegrity(pw, buf)
			if err != nil {
				log.Printf("Piece #%d failed integrity check", pw.index)
				workQueue <- pw
				continue
			}

			c.SendHave(pw.index)
			results <- &pieceResult{buf, pw.index}
		}
	}
}

func (t *Torrent) boundsForPiece(index int) (begin, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return
}

func (t *Torrent) pieceSize(index int) int {
	begin, end := t.boundsForPiece(index)
	return end - begin
}

// Download downloads the torrent and returns the complete file as bytes
func (t *Torrent) Download(ctx context.Context) ([]byte, error) {
	log.Println("Starting download for", t.Name)

	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)

	for index, hash := range t.PieceHashes {
		length := t.pieceSize(index)
		workQueue <- &pieceWork{index, hash, length}
	}

	for _, p := range t.Peers {
		go t.startWorker(ctx, p, workQueue, results)
	}

	buf := make([]byte, t.Length)
	donePieces := 0

	for donePieces < len(t.PieceHashes) {
		select {
		case <-ctx.Done():
			close(workQueue)
			log.Println("Download cancelled")
			return nil, ctx.Err()
		case res := <-results:
			begin, end := t.boundsForPiece(res.index)
			copy(buf[begin:end], res.buffer)
			donePieces++

			percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
			numWorkers := runtime.NumGoroutine() - 1
			log.Printf("[%.2f%%] Piece #%d | Peers: %d", percent, res.index, numWorkers)
		}
	}

	close(workQueue)
	return buf, nil
}
