package main

import (
	"btc/internal/config"
	"btc/internal/logger"
	"btc/internal/torrent"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Initialize logger (empty string = stderr, or provide path for file logging)
	if err := logger.Init(""); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChannel
		logger.Info("shutting down due to signal")
		cancel()
	}()

	// Validate arguments
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <torrent-file> <output-path>\n", os.Args[0])
		os.Exit(1)
	}

	cfg := config.Default()
	inPath := os.Args[1]
	outPath := os.Args[2]

	// Parse torrent file
	tf, err := torrent.Open(inPath)
	if err != nil {
		logger.Error("failed to open torrent", "error", err)
		os.Exit(1)
	}

	logger.Info("torrent loaded", "name", tf.Name, "size", tf.Length, "pieces", len(tf.PieceHashes))

	// Set up download options with progress callback
	opts := &torrent.DownloadOptions{
		OnProgress: func(percent float64, pieceIndex int, peerCount int) {
			fmt.Printf("\r[%.2f%%] Piece #%d | Peers: %d", percent, pieceIndex, peerCount)
		},
		OnEvent: func(event string, data map[string]any) {
			logger.Debug("event", "type", event, "data", data)
		},
	}

	// Download
	err = tf.DownloadToFile(ctx, outPath, cfg, opts)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("download interrupted")
		} else {
			logger.Error("download failed", "error", err)
			os.Exit(1)
		}
	}

	fmt.Println() // New line after progress
	logger.Info("download complete", "output", outPath)
}
