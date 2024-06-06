package ssz

import (
	"encoding/binary"
	"fmt"
	"math/bits"

	"github.com/prysmaticlabs/gohashtree"
)

const BYTES_PER_CHUNK = 32
const BYTES_PER_LENGTH_OFFSET = 4
const BITS_PER_BYTE = 8

func isBasicType(t SSZType) bool {
	switch t.(type) {
	case *BoolType, *UintType:
		return true
	}
	return false
}

func wrapError(t SSZType, err error) error {
	return fmt.Errorf("%s: %w", t.Name(), err)
}

func wrapErrorMsg(t SSZType, msg string) error {
	return fmt.Errorf("%s: %s", t.Name(), msg)
}

func chunksToDepth(n uint) uint {
	if n == 0 {
		return 0
	}
	return uint(bits.Len(n - 1))
}

func sizeToChunks(size uint) uint {
	return (size + 31) / 32
}

func Pack(b []byte) [][32]byte {
	newLen := (len(b) + 31) / 32
	packed := make([][32]byte, newLen)

	for i := 0; i < newLen; i++ {
		chunk := packed[i]
		copy(chunk[:], b[i*32:])
		packed[i] = chunk
	}
	return packed
}

func Hash64(left, right [32]byte) [32]byte {
	o := make([][32]byte, 1)
	gohashtree.HashChunks(o, [][32]byte{left, right})
	return o[0]
}

var zeroHashes = [][32]byte{
	{0x00},
}

func ZeroHash(depth uint) [32]byte {
	if depth >= uint(len(zeroHashes)) {
		for i := uint(len(zeroHashes)); i <= depth; i++ {
			zeroHashes = append(zeroHashes, Hash64(ZeroHash(i-1), ZeroHash(i-1)))
		}
	}
	return zeroHashes[depth]
}

func Merkleize(chunks [][32]byte, limit uint) [32]byte {
	layerCount := uint(bits.Len(limit - 1))
	if len(chunks) == 0 {
		return ZeroHash(layerCount)
	}

	for i := uint(0); i < layerCount; i++ {
		chunkCount := uint(len(chunks))
		if chunkCount%2 == 1 {
			chunks = append(chunks, ZeroHash(i))
			chunkCount++
		}

		nextChunks := make([][32]byte, chunkCount/2)
		gohashtree.HashChunks(nextChunks, chunks)

		chunks = nextChunks
	}

	return chunks[0]
}

func MixInLength(root [32]byte, length uint) [32]byte {
	var lengthBytes [32]byte
	binary.LittleEndian.PutUint64(lengthBytes[:], uint64(length))
	return Hash64(root, lengthBytes)
}
