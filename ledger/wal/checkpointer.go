package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/optakt/flow-dps/ledger/forest/flattener"
)

// FIXME: Cleanup.

const MagicBytes uint16 = 0x2137
const VersionV1 uint16 = 0x01

// Versions was reset while changing trie format, so now bump it to 3 to avoid conflicts
// Version 3 contains a file checksum for detecting corrupted checkpoint files.
const VersionV3 uint16 = 0x03

func ReadCheckpoint(r io.Reader) (*flattener.FlattenedForest, error) {

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
		return nil, fmt.Errorf("unsupported file version %x ", version)
	}

	if version != VersionV3 {
		reader = bufReader //switch back to plain reader
	}

	nodes := make([]*flattener.StorableNode, nodesCount+1) //+1 for 0 index meaning nil
	tries := make([]*flattener.StorableTrie, triesCount)

	for i := uint64(1); i <= nodesCount; i++ {
		storableNode, err := flattener.ReadStorableNode(reader)
		if err != nil {
			return nil, fmt.Errorf("cannot read storable node %d: %w", i, err)
		}
		nodes[i] = storableNode
	}

	// TODO version ?
	for i := uint16(0); i < triesCount; i++ {
		storableTrie, err := flattener.ReadStorableTrie(reader)
		if err != nil {
			return nil, fmt.Errorf("cannot read storable trie %d: %w", i, err)
		}
		tries[i] = storableTrie
	}

	if version == VersionV3 {
		crc32buf := make([]byte, 4)
		_, err := bufReader.Read(crc32buf)
		if err != nil {
			return nil, fmt.Errorf("error while reading CRC32 checksum: %w", err)
		}
		readCrc32, _ := readUint32(crc32buf, 0)

		calculatedCrc32 := crcReader.Crc32()

		if calculatedCrc32 != readCrc32 {
			return nil, fmt.Errorf("checkpoint checksum failed! File contains %x but read data checksums to %x", readCrc32, calculatedCrc32)
		}
	}

	return &flattener.FlattenedForest{
		Nodes: nodes,
		Tries: tries,
	}, nil

}

func readUint16(buffer []byte, location int) (uint16, int) {
	value := binary.BigEndian.Uint16(buffer[location:])
	return value, location + 2
}

func readUint32(buffer []byte, location int) (uint32, int) {
	value := binary.BigEndian.Uint32(buffer[location:])
	return value, location + 4
}

func readUint64(buffer []byte, location int) (uint64, int) {
	value := binary.BigEndian.Uint64(buffer[location:])
	return value, location + 8
}
