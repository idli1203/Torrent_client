package main

import (
	"btc/internal/config"
	"btc/internal/torrent"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	// adding context for handling cancelation and sigterm issues.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// signal handling
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChannel
		log.Println("Shutting down : SIGTERM error received")
		cancel()
	}()

	cfg := config.Default()

	inPath := os.Args[1]
	outPath := os.Args[2]

	tf, err := torrent.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadToFile(ctx, outPath, cfg)
	if err != nil {
		log.Fatal(err)
	}
}
