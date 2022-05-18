package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/rs/zerolog"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/ledger/forest"
	"github.com/optakt/flow-dps/ledger/trie"
)

// Magic number constants for Flow trie checkpoints.
const (
	MagicBytes uint16 = 0x2137

	VersionV1 uint16 = 0x01

	// VersionV3 is a file checksum for detecting corrupted checkpoint files.
	// The version was changed while updating trie format, so now it is set to 3 to avoid conflicts.
	VersionV3 uint16 = 0x03

	// VersionV4 contains a footer with node count and trie count (previously in the header).
	// Version 4 also reduces checkpoint data size.  See EncodeNode() and EncodeTrie() for more details.
	VersionV4 uint16 = 0x04

	// VersionV5 includes these changes:
	// - remove regCount and maxDepth from serialized nodes
	// - add allocated register count and size to serialized tries
	// - reduce number of bytes used to encode payload value size from 8 bytes to 4 bytes.
	// See EncodeNode() and EncodeTrie() for more details.
	VersionV5 uint16 = 0x05

	encMagicSize     = 2
	encVersionSize   = 2
	headerSize       = encMagicSize + encVersionSize
	encNodeCountSize = 8
	encTrieCountSize = 2
	crc32SumSize     = 4
)

// ReadCheckpoint reads a checkpoint and populates a store with its data, while
// also returning a light forest from the decoded data.
func ReadCheckpoint(log zerolog.Logger, r io.Reader) (*forest.LightForest, error) {

	var bufReader io.Reader = bufio.NewReader(r)
	crcReader := NewCRC32Reader(bufReader)
	var reader io.Reader = crcReader

	header := make([]byte, 4+8+2)

	_, err := io.ReadFull(reader, header)
	if err != nil {
		return nil, fmt.Errorf("cannot read header bytes: %w", err)
	}

	magicBytes, pos := readUint16(header, 0)
	version, pos := readUint16(header, pos)
	nodesCount, pos := readUint64(header, pos)
	triesCount, _ := readUint16(header, pos)

	if magicBytes != MagicBytes {
		return nil, fmt.Errorf("unknown file format. Magic constant %x does not match expected %x", magicBytes, MagicBytes)
	}
	if version != VersionV1 && version != VersionV3 {
		return nil, fmt.Errorf("unsupported file version %x", version)
	}

	if version < VersionV3 {
		// Switch to the plain reader.
		reader = bufReader
	}

	// The `nodes` slice needs one extra capacity for the nil node at index 0.
	nodes := make([]*trie.LightNode, nodesCount+1)
	tries := make([]*trie.LightTrie, triesCount)

	log.Info().
		Uint16("encoding_version", version).
		Uint64("nodes", nodesCount+1).
		Uint16("tries", triesCount).
		Msg("commencing checkpoint decoding")

	// Decode all light nodes.
	for i := uint64(1); i <= nodesCount; i++ {
		lightNode, err := trie.DecodeLightNode(reader)
		if err != nil {
			return nil, fmt.Errorf("could not read light node %d: %w", i, err)
		}
		nodes[i] = lightNode
	}

	// Decode all light tries.
	for i := uint16(0); i < triesCount; i++ {
		lightTrie, err := trie.DecodeLightTrie(reader)
		if err != nil {
			return nil, fmt.Errorf("could not read light trie %d: %w", i, err)
		}
		tries[i] = lightTrie
	}

	// If the checkpoint uses the version 3, we need to read its CRC32 checksum first,
	// and then to use the CRC reader to verify it.
	if version == VersionV3 {
		crc32buf := make([]byte, 4)
		_, err := bufReader.Read(crc32buf)
		if err != nil {
			return nil, fmt.Errorf("could not read CRC32 checksum: %w", err)
		}
		readCrc32, _ := readUint32(crc32buf, 0)

		calculatedCrc32 := crcReader.Crc32()

		if calculatedCrc32 != readCrc32 {
			return nil, fmt.Errorf("invalid CRC32 checksum: got %x want %x", readCrc32, calculatedCrc32)
		}
	}

	return &forest.LightForest{
		Nodes: nodes,
		Tries: tries,
	}, nil
}

// Checkpoint writes a CBOR-encoded and ZSTD-compressed trie as a checkpoint in the given writer.
func Checkpoint(writer io.Writer, trie *trie.Trie) error {

	// FIXME: Instead of doing this, iterate through all nodes and encode them with a simple format.
	encTrie, err := trie.Encode()
	if err != nil {
		return fmt.Errorf("could not encode trie: %w", err)
	}

	compressed, err := zbor.NewCodec().Compress(encTrie)
	if err != nil {
		return fmt.Errorf("could not compress trie: %w", err)
	}

	_, err = writer.Write(compressed)
	if err != nil {
		return fmt.Errorf("could not write encoded trie: %w", err)
	}

	return nil
}

func readUint16(buffer []byte, offset int) (value uint16, position int) {
	value = binary.BigEndian.Uint16(buffer[offset:])
	return value, offset + 2
}

func readUint32(buffer []byte, offset int) (value uint32, position int) {
	value = binary.BigEndian.Uint32(buffer[offset:])
	return value, offset + 4
}

func readUint64(buffer []byte, offset int) (value uint64, position int) {
	value = binary.BigEndian.Uint64(buffer[offset:])
	return value, offset + 8
}
