package torrent

import (
	"btc/internal/config"
	"btc/internal/download"
	"btc/internal/logger"
	"btc/internal/peer"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/jackpal/bencode-go"
)

// bencodeInfo contains the info dictionary from a .torrent file
type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
	PieceLength int    `bencode:"piece length"`
}

// bencodeTorrent represents a parsed .torrent file
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

// bencodeTrackerResp holds the tracker response
type bencodeTrackerResp struct {
	Peers    string `bencode:"peers"`
	Interval int    `bencode:"interval"`
}

// DownloadOptions configures the download behavior
type DownloadOptions struct {
	OnProgress download.ProgressCallback
	OnEvent    download.EventCallback
}

// TrackerURL builds the announce URL with required parameters
func (t *TorrentFile) TrackerURL(peerID [20]byte, port uint16) (string, error) {
	parsedURL, err := url.Parse(t.Announce)
	if err != nil {
		return "", fmt.Errorf("parsing tracker URL: %w", err)
	}

	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	parsedURL.RawQuery = params.Encode()

	return parsedURL.String(), nil
}

// RequestPeers contacts the tracker and returns a list of peers
func (t *TorrentFile) RequestPeers(peerID [20]byte, port uint16, cfg *config.Config) ([]peer.Peer, error) {
	trackerURL, err := t.TrackerURL(peerID, port)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: cfg.TrackerTimeout}
	resp, err := client.Get(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("contacting tracker: %w", err)
	}
	defer resp.Body.Close()

	var trackerResp bencodeTrackerResp
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, fmt.Errorf("parsing tracker response: %w", err)
	}

	return peer.UnmarshalPeers([]byte(trackerResp.Peers))
}

// DownloadToFile downloads the torrent and saves it to the specified path
func (t *TorrentFile) DownloadToFile(ctx context.Context, path string, cfg *config.Config, opts *DownloadOptions) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return fmt.Errorf("generating peer ID: %w", err)
	}

	logger.Info("requesting peers from tracker", "announce", t.Announce)
	peers, err := t.RequestPeers(peerID, 8080, cfg)
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

	buf, err := torrent.Download(ctx)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}

	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	_, err = outFile.Write(buf)
	if err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	logger.Info("file saved", "path", path)
	return nil
}

// Open parses a .torrent file and returns a TorrentFile
func Open(path string) (*TorrentFile, error) {
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
	err := bencode.Marshal(&buf, *info)
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
