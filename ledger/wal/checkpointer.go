package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

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
)

// ReadCheckpoint reads a checkpoint and populates a store with its data, while
// also returning a light forest from the decoded data.
func ReadCheckpoint(r io.Reader) (*forest.LightForest, error) {

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

	fmt.Println("Expect to read", nodesCount, "nodes and", triesCount, "tries")

	if magicBytes != MagicBytes {
		return nil, fmt.Errorf("unknown file format. Magic constant %x does not match expected %x", magicBytes, MagicBytes)
	}
	if version != VersionV1 && version != VersionV3 {
		return nil, fmt.Errorf("unsupported file version %x", version)
	}

	if version != VersionV3 {
		// Switch to the plain reader.
		reader = bufReader
	}

	// The `nodes` slice needs one extra capacity for the nil node at index 0.
	nodes := make([]*trie.LightNode, nodesCount+1)
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
