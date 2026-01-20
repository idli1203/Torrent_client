package config

import "time"

// Config holds all tunable parameters for the BitTorrent client
type Config struct {
	BlockSize        int
	HandshakeTimeout time.Duration
	TCPTimeout       time.Duration // Fixed: was TcpTimeout
	PieceTimeout     time.Duration
	TrackerTimeout   time.Duration
	RequestBacklog   int
}

// Default returns a Config with sensible default values
func Default() *Config {
	return &Config{
		BlockSize:        16 * 1024,
		HandshakeTimeout: 15 * time.Second,
		TCPTimeout:       15 * time.Second,
		PieceTimeout:     30 * time.Second,
		TrackerTimeout:   30 * time.Second,
		RequestBacklog:   50,
	}
}
