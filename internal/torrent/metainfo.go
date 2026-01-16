package torrent

import (
	"btc/internal/download"
	"btc/internal/peer"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

// BencodeInfo is the struct which contains the info dictionary and data about the torrent file
type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
	PieceLength int    `bencode:"piece length"`
}

// Bencoded torrent file depackaged to a struct
type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// more of a compact type of struct constructed in order to have all relavant data in one struct
type TorrentFile struct {
	Name        string
	Announce    string
	PieceHashes [][20]byte
	InfoHash    [20]byte
	PieceLength int
	Length      int
}

// bencode_tracker_resp holds the tracker response
type bencode_tracker_resp struct {
	Peers    string `bencode:"peers"`
	Interval int    `bencode:"interval"`
}

func (t *TorrentFile) TrackerURL(peerID [20]byte, port uint16) (string, error) {
	parsed_url, err := url.Parse(t.Announce)
	if err != nil {
		fmt.Println("URL could not be parsed")
		return "", err
	}
	parameters_url := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	parsed_url.RawQuery = parameters_url.Encode()

	return parsed_url.String(), nil
}

func (t *TorrentFile) RequestPeers(peerID [20]byte, port uint16) ([]peer.Peer, error) {
	trackerURL, err := t.TrackerURL(peerID, port)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Get(trackerURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	tracker_resp := bencode_tracker_resp{}
	err = bencode.Unmarshal(response.Body, &tracker_resp)
	if err != nil {
		return nil, err
	}
	return peer.Unmarshal_Peer([]byte(tracker_resp.Peers))
}

func (t *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	peers, err := t.RequestPeers(peerID, 8080)
	if err != nil {
		return err
	}

	torrent := download.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func Open(torrent_file string) (*TorrentFile, error) {
	file, err := os.Open(torrent_file)
	if err != nil {
		return nil, fmt.Errorf("failed to read the torrent file: %w", err)
	}

	var torrent bencodeTorrent

	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal the torrent file: %w", err)
	}

	return torrent.ToTorrentstruct()
}

func (tor_info *bencodeInfo) ComputeinfoHash() ([20]byte, error) {
	var buffer bytes.Buffer

	err := bencode.Marshal(&buffer, *tor_info)
	if err != nil {
		fmt.Println("The sha1 hash is not possible")
		return [20]byte{}, err
	}

	compute := sha1.Sum(buffer.Bytes())

	return compute, nil
}

func (tor_info *bencodeInfo) SplitToPieces() ([][20]byte, error) {
	sha1_hash_length := 20
	buf := []byte(tor_info.Pieces)

	if len(buf)%sha1_hash_length != 0 {
		return nil, fmt.Errorf("pieces are wrongly encoded, length %d not divisible by %d", len(buf), sha1_hash_length)
	}

	numPieces := len(buf) / sha1_hash_length

	fmt.Println("the number of Pieces are : ", numPieces)

	PieceHashes := make([][20]byte, numPieces)
	for i := 0; i < numPieces; i++ {
		copy(PieceHashes[i][:], tor_info.Pieces[i*sha1_hash_length:(i+1)*sha1_hash_length])
	}
	return PieceHashes, nil
}

func (torrent bencodeTorrent) ToTorrentstruct() (*TorrentFile, error) {
	all_pieces, err := torrent.Info.SplitToPieces()
	if err != nil {
		return nil, err
	}
	info_hashed, err := torrent.Info.ComputeinfoHash()
	if err != nil {
		return nil, err
	}
	return &TorrentFile{
		Name:        torrent.Info.Name,
		Announce:    torrent.Announce,
		PieceHashes: all_pieces,
		InfoHash:    info_hashed,
		PieceLength: torrent.Info.PieceLength,
		Length:      torrent.Info.Length,
	}, nil
}
