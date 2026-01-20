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

	// SIGTERM and SIGINT handle using single unit channel
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChannel
		log.Println("Shutting down due to SIGTERM/SIGINT error")
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
