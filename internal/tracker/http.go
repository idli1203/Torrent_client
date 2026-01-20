package tracker

import (
	"btc/internal/config"
	"btc/internal/peer"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/jackpal/bencode-go"
)

// bencodeTrackerResp holds the tracker response
type bencodeTrackerResp struct {
	Peers    string `bencode:"peers"`
	Interval int    `bencode:"interval"`
}

// HTTPTracker implements the Tracker interface for HTTP/HTTPS trackers
type HTTPTracker struct {
	AnnounceURL string
	Cfg         *config.Config
}

// NewHTTPTracker creates a new HTTP tracker client
func NewHTTPTracker(announceURL string, cfg *config.Config) *HTTPTracker {
	return &HTTPTracker{
		AnnounceURL: announceURL,
		Cfg:         cfg,
	}
}

// BuildURL constructs the announce URL with required parameters
func (t *HTTPTracker) BuildURL(peerID [20]byte, port uint16, infoHash [20]byte, left int) (string, error) {
	parsedURL, err := url.Parse(t.AnnounceURL)
	if err != nil {
		return "", fmt.Errorf("parsing tracker URL: %w", err)
	}

	params := url.Values{
		"info_hash":  []string{string(infoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(left)},
	}
	parsedURL.RawQuery = params.Encode()

	return parsedURL.String(), nil
}

// Announce contacts the tracker and returns a list of peers
func (t *HTTPTracker) Announce(peerID [20]byte, port uint16, infoHash [20]byte, left int) ([]peer.Peer, error) {
	announceURL, err := t.BuildURL(peerID, port, infoHash, left)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: t.Cfg.TrackerTimeout}
	resp, err := client.Get(announceURL)
	if err != nil {
		return nil, fmt.Errorf("contacting tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker returned status %d", resp.StatusCode)
	}

	var trackerResp bencodeTrackerResp
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, fmt.Errorf("parsing tracker response: %w", err)
	}

	return peer.UnmarshalPeers([]byte(trackerResp.Peers))
}

// Ensure HTTPTracker implements Tracker interface
var _ Tracker = (*HTTPTracker)(nil)
