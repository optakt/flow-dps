package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/rs/zerolog"

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

// StoreCheckpoint writes the given tries to checkpoint file, and also appends
// a CRC32 file checksum for integrity check.
// Checkpoint file consists of a flattened forest. Specifically, it consists of:
//   * a list of encoded nodes, where references to other nodes are by list index.
//   * a list of encoded tries, each referencing their respective root node by index.
// Referencing to other nodes by index 0 is a special case, meaning nil.
//
// As an important property, the nodes are listed in an order which satisfies
// Descendents-First-Relationship. The Descendents-First-Relationship has the
// following important property:
// When rebuilding the trie from the sequence of nodes, build the trie on the fly,
// as for each node, the children have been previously encountered.
func StoreCheckpoint(writer io.Writer, trie *trie.Trie) error {

	crc32Writer := NewCRC32Writer(writer)

	// Scratch buffer is used as temporary buffer that node can encode into.
	// Data in scratch buffer should be copied or used before scratch buffer is used again.
	// If the scratch buffer isn't large enough, a new buffer will be allocated.
	// However, 4096 bytes will be large enough to handle almost all payloads
	// and 100% of interim nodes.
	scratch := make([]byte, 1024*4)

	// Write header: magic (2 bytes) + version (2 bytes)
	header := scratch[:headerSize]
	binary.BigEndian.PutUint16(header, MagicBytes)
	binary.BigEndian.PutUint16(header[encMagicSize:], VersionV5)

	_, err := crc32Writer.Write(header)
	if err != nil {
		return fmt.Errorf("cannot write checkpoint header: %w", err)
	}

	// Serialize all unique nodes
	trieBytes, err := trie.Encode()
	if err != nil {
		return fmt.Errorf("could not encode trie: %w", err)
	}

	crc32Writer.Write(trieBytes)

	// Write CRC32 sum
	crc32buf := scratch[:crc32SumSize]
	binary.BigEndian.PutUint32(crc32buf, crc32Writer.Crc32())

	_, err = writer.Write(crc32buf)
	if err != nil {
		return fmt.Errorf("cannot write CRC32: %w", err)
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
