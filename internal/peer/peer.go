package peer

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func UnmarshalPeers(peerData []byte) ([]Peer, error) {
	const peerSize = 6

	if len(peerData)%peerSize != 0 {
		return nil, fmt.Errorf("invalid peer list length: %d not divisible by %d", len(peerData), peerSize)
	}

	numPeers := len(peerData) / peerSize
	peers := make([]Peer, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peerData[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peerData[offset+4 : offset+6])
	}

	return peers, nil
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}
