package main

import (
	"btc/internal/config"
	"btc/internal/torrent"
	"log"
	"os"
)

func main() {
	cfg := config.Default()
	inPath := os.Args[1]
	outPath := os.Args[2]

	tf, err := torrent.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadToFile(outPath, cfg)
	if err != nil {
		log.Fatal(err)
	}
}
