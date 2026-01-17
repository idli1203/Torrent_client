package config

import time

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

