package bitfield

type Bitfield []byte

func (bf Bitfield) HasPiece(index int) bool {
	byteindex := index / 8
	bitindex := index % 8

	return bf[byteindex]&(1<<(7-bitindex)) != 0
}

func (bf Bitfield) SetPiece(index int) {
	byteindex := index / 8
	bitindex := index % 8

	bf[byteindex] |= (1 << (7 - bitindex))
}
