package torrent

import (
	"btc/internal/config"
	"btc/internal/download"
	"btc/internal/logger"
	"btc/internal/tracker"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/jackpal/bencode-go"
)

// Info dict struct
type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
	PieceLength int    `bencode:"piece length"`
}

// Represents a .torrent file (Only relevant parameters)
type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// TorrentFile contains processed torrent metadata
type TorrentFile struct {
	Name        string
	Announce    string
	PieceHashes [][20]byte
	InfoHash    [20]byte
	PieceLength int
	Length      int
}

// For wiring progress and events tracking into ui
type DownloadOptions struct {
	OnProgress download.ProgressCallback
	OnEvent    download.EventCallback
}

// DownloadToFile downloads the torrent and saves it to the specified path
func (t *TorrentFile) DownloadToFile(ctx context.Context, path string, cfg *config.Config, opts *DownloadOptions) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return fmt.Errorf("generating peer ID: %w", err)
	}

	logger.Info("requesting peers from tracker", "announce", t.Announce)
	httpTracker := tracker.NewHTTPTracker(t.Announce, cfg)
	peers, err := httpTracker.Announce(peerID, 8080, t.InfoHash, t.Length)
	if err != nil {
		return fmt.Errorf("requesting peers: %w", err)
	}
	logger.Info("received peers", "count", len(peers))

	torrent := download.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
		Cfg:         cfg,
	}

	if opts != nil {
		torrent.OnProgress = opts.OnProgress
		torrent.OnEvent = opts.OnEvent
	}

	err = torrent.Download(ctx, path)
	if err != nil {
		logger.Error(`Issue while downloading file: ` + err.Error())
		return err
	}

	logger.Info("file saved", "path", path)
	return nil
}

// Open parses a .torrent file and returns a TorrentFile
func Open(path string) (*TorrentFile, error) {
	logger.Info("opening torrent file", "path", path)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening torrent file: %w", err)
	}
	defer file.Close()

	var bto bencodeTorrent
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return nil, fmt.Errorf("parsing torrent file: %w", err)
	}

	return bto.ToTorrentFile()
}

// ComputeInfoHash calculates the SHA1 hash of the info dictionary
func (info *bencodeInfo) ComputeInfoHash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, info)
	if err != nil {
		return [20]byte{}, fmt.Errorf("marshaling info dict: %w", err)
	}
	return sha1.Sum(buf.Bytes()), nil
}

// SplitPieceHashes splits the pieces string into individual hashes
func (info *bencodeInfo) SplitPieceHashes() ([][20]byte, error) {
	const hashLen = 20
	buf := []byte(info.Pieces)

	if len(buf)%hashLen != 0 {
		return nil, fmt.Errorf("malformed pieces: length %d not divisible by %d", len(buf), hashLen)
	}

	numPieces := len(buf) / hashLen
	hashes := make([][20]byte, numPieces)

	for i := 0; i < numPieces; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}

	return hashes, nil
}

// ToTorrentFile converts a bencodeTorrent to a TorrentFile
func (bto *bencodeTorrent) ToTorrentFile() (*TorrentFile, error) {
	pieceHashes, err := bto.Info.SplitPieceHashes()
	if err != nil {
		return nil, err
	}

	infoHash, err := bto.Info.ComputeInfoHash()
	if err != nil {
		return nil, err
	}

	return &TorrentFile{
		Name:        bto.Info.Name,
		Announce:    bto.Announce,
		PieceHashes: pieceHashes,
		InfoHash:    infoHash,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
	}, nil
}
