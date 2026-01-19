package tracker

import "btc/internal/peer"

// Tracker defines the interface for tracker communication.
// This allows swapping between HTTP and UDP tracker implementations.
type Tracker interface {
	// Announce contacts the tracker and returns a list of peers
	Announce(peerID [20]byte, port uint16, infoHash [20]byte, left int) ([]peer.Peer, error)
}

// HTTPTracker implements the Tracker interface for HTTP trackers
// (implementation in http.go when needed)

// UDPTracker implements the Tracker interface for UDP trackers
// (implementation in udp.go when needed)
