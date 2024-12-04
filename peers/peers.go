package peers

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func Unmarshal_Peer(peer_collection []byte) ([]Peer, error) {
	// Peer_collection
	const Peersize = 6

	numofpeers := len(peer_collection) / Peersize

	Peers := make([]Peer, numofpeers)

	if len(peer_collection)%Peersize != 0 {
		log.Fatal("The given peer list is error prone or incomplete.")

		return nil, fmt.Errorf("%s", "Peer list incomplete")
	}

	for i := 0; i < numofpeers; i++ {
		offset := i * Peersize

		Peers[i].IP = net.IP(peer_collection[offset : offset+4])
		Peers[i].Port = binary.BigEndian.Uint16([]byte(peer_collection[offset+4 : offset+6]))
	}

	return Peers, nil
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}
