package protocol

// Bitfield represents the pieces a peer has
type Bitfield []byte

// HasPiece checks if a piece is available in the bitfield
func (bf Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	bitIndex := index % 8

	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}

	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
}

// SetPiece marks a piece as available in the bitfield
func (bf Bitfield) SetPiece(index int) {
	byteIndex := index / 8
	bitIndex := index % 8

	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}

	bf[byteIndex] |= 1 << (7 - bitIndex)
}
