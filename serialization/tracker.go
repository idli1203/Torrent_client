package serialization

import (
	tracker "bitorrent_try/peers"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

type bencode_tracker_resp struct {
	Peers    string `bencode:"peers"`
	Interval int    `bencode:"interval"`
}

func (torrent_file *TorrentFile) TrackerURL(peerID [20]byte, port uint16) (string, error) {
	parsed_url, err := url.Parse(torrent_file.Announce)
	if err != nil {
		fmt.Println("URL could not be parsed")
		return "", err
	}
	parameters_url := url.Values{
		"info_hash":  []string{string(torrent_file.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{string(strconv.Itoa(int(port)))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(torrent_file.Length)},
	}
	parsed_url.RawQuery = parameters_url.Encode()

	return parsed_url.String(), nil
}

func (torrent *TorrentFile) RequestPeers(PeerID [20]byte, port uint16) ([]tracker.Peer, error) {
	url, err := torrent.TrackerURL(PeerID, port)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	tracker_resp := bencode_tracker_resp{}
	err = bencode.Unmarshal(response.Body, &tracker_resp)
	if err != nil {
		return nil, err
	}
	return tracker.Unmarshal_Peer([]byte(tracker_resp.Peers))
}
