package serialization

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"os"

	"github.com/jackpal/bencode-go"
)

// BencodeInfo is the struct which contains the info dictionary and data about the torrent file.
type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
	PieceLength int    `bencode:"piece length"`
}

// Bencoded .torrent file depackaged to a struct
type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// more of a compact type of struct constructed in order to have all relavant data in one struct
type TorrentFile struct {
	Name        string
	Announce    string
	PieceHashes [][20]byte
	InfoHash    [20]byte
	PieceLength int
	Length      int
}

func Open(torrent_file string) (*bencodeTorrent, error) {
	file, err := os.Open(torrent_file)
	if err != nil {
		log.Fatal("Failed to read the torrent file : ", err)
	}

	var torrent bencodeTorrent

	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		log.Fatal("could not unmarshal the torrent_file", err)
	}

	return &torrent, nil
}

func (tor_info *bencodeInfo) ComputeinfoHash() ([20]byte, error) {
	var buffer bytes.Buffer

	err := bencode.Marshal(&buffer, *tor_info)
	if err != nil {
		fmt.Println("The sha1 hash is not possible")
		return [20]byte{}, err
	}

	// fmt.Println("serialize_info : ", buffer.String())

	compute := sha1.Sum(buffer.Bytes())

	return compute, nil
}

func (tor_info *bencodeInfo) SplitToPieces() ([][20]byte, error) {
	sha1_hash_length := 20
	buf := []byte(tor_info.Pieces)

	if len(buf)%sha1_hash_length != 0 {
		log.Fatal("The Pieces are wrongly encoded , please check the torrent file ")

		return nil, fmt.Errorf("error in function splittopieces %d", 12)
	}

	numPieces := len(buf) / sha1_hash_length

	fmt.Println("the number of Pieces are : ", numPieces)

	PieceHashes := make([][20]byte, numPieces)
	for i := 0; i < numPieces; i++ {
		copy(PieceHashes[i][:], tor_info.Pieces[i*sha1_hash_length:(i+1)*sha1_hash_length])
	}
	return PieceHashes, nil
}

func (torrent bencodeTorrent) ToTorrentstruct() *TorrentFile {
	all_pieces, _ := torrent.Info.SplitToPieces()
	info_hashed, _ := torrent.Info.ComputeinfoHash()
	return &TorrentFile{
		Name:        torrent.Info.Name,
		Announce:    torrent.Announce,
		PieceHashes: all_pieces,
		InfoHash:    info_hashed,
		PieceLength: torrent.Info.PieceLength,
		Length:      torrent.Info.Length,
	}
}
