package config

import "time"

// config types : blocksize , handshake tout , tcp tout , piece tout , tracker tout , max peerval , request backlog (pipeline vala)

type Config struct {
	 BlockSize     int
	 HandshakeTimeout time.Duration
	 TcpTimeout       time.Duration
	 PieceTimeout     time.Duration
	 TrackerTimeout   time.Duration
	 MaxPeers      int
	 RequestBacklog int
}

func Default() *Config {

	return &Config{
		BlockSize:        16 * 1024,
		HandshakeTimeout: 15 * time.Second,
		TcpTimeout:       15 * time.Second,
		PieceTimeout:     30 * time.Second,
		TrackerTimeout:   30 * time.Second,
		MaxPeers:         50,
		RequestBacklog:   50,
	}

}
