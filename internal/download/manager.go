package download

import (
	"btc/internal/config"
	"btc/internal/logger"
	"btc/internal/peer"
	"btc/internal/protocol"
	"btc/internal/stats"
	"btc/internal/storage"
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"runtime"
	"time"
)

// ProgressCallback is called to report download progress
type ProgressCallback func(percent float64, pieceIndex int, peerCount int, speed float64)

// EventCallback is called to report events during download
type EventCallback func(event string, data map[string]any)

var outputpath string

type Torrent struct {
	PieceHashes [][20]byte
	Name        string
	Peers       []peer.Peer
	Length      int
	PieceLength int
	PeerID      [20]byte
	InfoHash    [20]byte
	Cfg         *config.Config
	rateCalc    *stats.RateCalculator
	OnProgress  ProgressCallback
	OnEvent     EventCallback
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

func (state *pieceProgress) ReadMessage() error {
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

func (t *Torrent) DownloadPiece(c *peer.Client, pw *pieceWork) ([]byte, error) {
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
					return nil, fmt.Errorf("sending request for piece %d: %w", pw.index, err)
				}

				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("reading message for piece %d: %w", pw.index, err)
		}
	}

	return state.buf, nil
}

func CheckIntegrity(pw *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], pw.hash[:]) {
		return fmt.Errorf("piece %d failed integrity check", pw.index)
	}
	return nil
}

func (t *Torrent) emitEvent(event string, data map[string]any) {
	if t.OnEvent != nil {
		t.OnEvent(event, data)
	}
}

func (t *Torrent) StartWorker(ctx context.Context, p peer.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := peer.New(p, t.PeerID, t.InfoHash, t.Cfg)
	if err != nil {
		logger.Debug("handshake failed", "peer", p.IP.String(), "error", err)
		t.emitEvent("handshake_failed", map[string]any{"peer": p.IP.String(), "error": err.Error()})
		return
	}
	defer c.Close()

	logger.Debug("handshake successful", "peer", p.IP.String())
	t.emitEvent("handshake_success", map[string]any{"peer": p.IP.String()})

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

			buf, err := t.DownloadPiece(c, pw)
			if err != nil {
				logger.Debug("piece download failed", "piece", pw.index, "error", err)
				workQueue <- pw
				return
			}

			err = CheckIntegrity(pw, buf)
			if err != nil {
				logger.Debug("integrity check failed", "piece", pw.index)
				workQueue <- pw
				continue
			}

			c.SendHave(pw.index)
			results <- &pieceResult{buf, pw.index}
		}
	}
}

func (t *Torrent) BoundsForPiece(index int) (begin, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return
}

func (t *Torrent) PieceSize(index int) int {
	begin, end := t.BoundsForPiece(index)
	return end - begin
}

// Download downloads the torrent and returns the complete file as bytes
func (t *Torrent) Download(ctx context.Context) ([]byte, error) {
	logger.Info("starting download", "name", t.Name, "size", t.Length, "pieces", len(t.PieceHashes))

	completedPieces := make([]bool, len(t.PieceHashes))

	t.rateCalc = stats.NewRateCalculator(1 * time.Second)
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)

	for index, hash := range t.PieceHashes {
		length := t.PieceSize(index)
		workQueue <- &pieceWork{index, hash, length}
	}

	for _, p := range t.Peers {
		go t.StartWorker(ctx, p, workQueue, results)
	}

	buf := make([]byte, t.Length)
	donePieces := 0

	for donePieces < len(t.PieceHashes) {
		select {
		case <-ctx.Done():
			close(workQueue)
			logger.Info("download cancelled")
			resumePath := outputpath + ".resume"
			storage.SaveResume(resumePath, &storage.ResumeData{
				InfoHash:        t.InfoHash,
				CompletedPieces: completedPieces,
				DownloadedBytes: int64(donePieces * t.PieceLength),
			})
			return nil, ctx.Err()
		case res := <-results:
			begin, end := t.BoundsForPiece(res.index)
			copy(buf[begin:end], res.buffer)
			completedPieces[res.index] = true
			donePieces++

			t.rateCalc.Add(int64(len(res.buffer)))
			percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
			numWorkers := runtime.NumGoroutine() - 1

			// Use callback instead of direct logging
			if t.OnProgress != nil {
				speed := t.rateCalc.Rate()
				t.OnProgress(percent, res.index, numWorkers, speed)
			}

			logger.Debug("piece downloaded", "piece", res.index, "percent", percent)
		}
	}

	close(workQueue)
	logger.Info("download complete", "name", t.Name)
	return buf, nil
}
