package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"

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

	VersionDPSV1 uint16 = 0xFF01

	encMagicSize     = 2
	encVersionSize   = 2
	headerSize       = encMagicSize + encVersionSize
	encNodeCountSize = 8
	encTrieCountSize = 2
	crc32SumSize     = 4
)

// defaultBufioReadSize replaces the default bufio buffer size of 4096 bytes.
// defaultBufioReadSize can be increased to 8KiB, 16KiB, 32KiB, etc. if it
// improves performance on typical EN hardware.
const defaultBufioReadSize = 1024 * 32

// ReadCheckpoint reads a checkpoint and populates a store with its data, while
// also returning a light forest from the decoded data.
func ReadCheckpoint(f *os.File) (*forest.LightForest, error) {

	header := make([]byte, headerSize)
	_, err := io.ReadFull(f, header)
	if err != nil {
		return nil, fmt.Errorf("cannot read header: %w", err)
	}

	// Decode header
	magicBytes := binary.BigEndian.Uint16(header)
	version := binary.BigEndian.Uint16(header[encMagicSize:])

	// Reset offset
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("cannot seek to start of file: %w", err)
	}

	if magicBytes != MagicBytes {
		return nil, fmt.Errorf("unknown file format. Magic constant %x does not match expected %x", magicBytes, MagicBytes)
	}

	var forest *forest.LightForest
	switch version {

	case VersionDPSV1:
		forest, err = readOptimizedCheckpoint(f)
		if err != nil {
			return nil, fmt.Errorf("cannot decode optimized format: %w", err)
		}

	case VersionV1, VersionV3:
		forest, err = readCheckpointV3AndEarlier(f, version)
		if err != nil {
			return nil, fmt.Errorf("cannot decode original format: %w", err)
		}

	case VersionV4:
		return nil, fmt.Errorf("FIXME version %d", version)

	case VersionV5:
		return nil, fmt.Errorf("FIXME version %d", version)

	default:
		return nil, fmt.Errorf("unsupported checkpoint version %x", version)

	}

	return forest, nil
}

func readCheckpointV3AndEarlier(f *os.File, version uint16) (*forest.LightForest, error) {
	var bufReader io.Reader = bufio.NewReaderSize(f, defaultBufioReadSize)
	crcReader := NewCRC32Reader(bufReader)

	var reader io.Reader
	if version != VersionV3 {
		reader = bufReader
	} else {
		reader = crcReader
	}

	// Read header (magic + version), node count, and trie count.
	header := make([]byte, headerSize+encNodeCountSize+encTrieCountSize)

	_, err := io.ReadFull(reader, header)
	if err != nil {
		return nil, fmt.Errorf("cannot read header: %w", err)
	}

	// Magic and version are verified by the caller.

	// Decode node count and trie count
	nodesCount := binary.BigEndian.Uint64(header[headerSize:])
	triesCount := binary.BigEndian.Uint16(header[headerSize+encNodeCountSize:])

	nodes := make([]*trie.LightNode, nodesCount+1) // +1 for 0 index meaning nil
	tries := make([]*trie.LightTrie, triesCount)

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

	if version == VersionV3 {
		crc32buf := make([]byte, crc32SumSize)

		_, err := io.ReadFull(bufReader, crc32buf)
		if err != nil {
			return nil, fmt.Errorf("cannot read CRC32: %w", err)
		}

		readCrc32 := binary.BigEndian.Uint32(crc32buf)

		calculatedCrc32 := crcReader.Crc32()

		if calculatedCrc32 != readCrc32 {
			return nil, fmt.Errorf("checkpoint checksum failed! File contains %x but calculated crc32 is %x", readCrc32, calculatedCrc32)
		}
	}

	forest := forest.LightForest{
		Nodes: nodes,
		Tries: tries,
	}

	return &forest, nil
}

func readOptimizedCheckpoint(f *os.File) (*forest.LightForest, error) {
	// FIXME: to do
	//nodes, err := trie.Decode(reader)

	return nil, nil
}

// Checkpoint writes a CBOR-encoded and ZSTD-compressed trie as a checkpoint in the given writer.
func Checkpoint(writer io.Writer, trie *trie.Trie) error {

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
