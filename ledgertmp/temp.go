package ledgertmp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/rs/zerolog"
)

const (
	crc32SumSize         = 4
	defaultBufioReadSize = 1024 * 32
	subtrieLevel         = 4
	subtrieCount         = 1 << subtrieLevel // 16
	encNodeCountSize     = 8
	encSubtrieCountSize  = 2
	encMagicSize         = 2
	encVersionSize       = 2
	headerSize           = encMagicSize + encVersionSize
)

var ErrEOFNotReached = errors.New("expect to reach EOF, but actually didn't")

// TODO: Move this type def into flow-go/ledger
type LeafNode struct {
	Hash    hash.Hash
	Path    ledger.Path
	Payload *ledger.Payload
}

// TODO: Move this type def into flow-go/ledger
type ReadingResult struct {
	LeafNode *LeafNode
	Err      error
}

// TODO: Move this into flow-go/ledger
func ReadLeafNodeFromCheckpoint(dir string, fileName string, logger *zerolog.Logger) (<-chan *ReadingResult, error) {
	// if checkpoint file not exist, return error
	// otherwise, read the leaf nodes from the given checkpoint and pass to resultChan
	resultChan := make(chan *ReadingResult)
	err := readLeafNodes(dir, fileName, resultChan, logger)
	if err != nil {
		logger.Err(err).Msg("failed to read leaf nodes from checkpoint file")
		return nil, fmt.Errorf("failed to read leaf nodes from checkpoint file: %w", err)
	}
	return resultChan, nil
}

// readCheckpointHeader takes a file path and returns subtrieChecksums and topTrieChecksum
// any error returned are exceptions
func readCheckpointHeader(filepath string, logger *zerolog.Logger) (
	checksumsOfSubtries []uint32,
	checksumOfTopTrie uint32,
	errToReturn error,
) {
	closable, err := os.Open(filepath)
	if err != nil {
		return nil, 0, fmt.Errorf("could not open header file: %w", err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Err(err)
		}
	}(closable)

	var bufReader io.Reader = bufio.NewReaderSize(closable, defaultBufioReadSize)
	reader := wal.NewCRC32Reader(bufReader)
	// read the magic bytes and check version
	err = validateFileHeader(wal.MagicBytesCheckpointHeader, wal.VersionV6, reader)
	if err != nil {
		return nil, 0, err
	}

	// read the subtrie count
	subtrieCount, err := readSubtrieCount(reader)
	if err != nil {
		return nil, 0, err
	}

	subtrieChecksums := make([]uint32, subtrieCount)
	for i := uint16(0); i < subtrieCount; i++ {
		sum, err := readCRC32Sum(reader)
		if err != nil {
			return nil, 0, fmt.Errorf("could not read %v-th subtrie checksum from checkpoint header: %w", i, err)
		}
		subtrieChecksums[i] = sum
	}

	// read top level trie checksum
	topTrieChecksum, err := readCRC32Sum(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("could not read checkpoint top level trie checksum in chechpoint summary: %w", err)
	}

	// calculate the actual checksum
	actualSum := reader.Crc32()

	// read the stored checksum, and compare with the actual sum
	expectedSum, err := readCRC32Sum(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("could not read checkpoint header checksum: %w", err)
	}

	if actualSum != expectedSum {
		return nil, 0, fmt.Errorf("invalid checksum in checkpoint header, expected %v, actual %v",
			expectedSum, actualSum)
	}

	err = ensureReachedEOF(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("fail to read checkpoint header file: %w", err)
	}

	return subtrieChecksums, topTrieChecksum, nil
}

func readLeafNodes(dir string, fileName string, result chan<- *ReadingResult, logger *zerolog.Logger) error {
	headerPath := filepath.Join(dir, fileName)
	lg := logger.With().Str("checkpoint_file", headerPath).Logger()
	lg.Info().Msgf("reading v6 checkpoint file")

	subtrieChecksums, _, err := readCheckpointHeader(headerPath, logger)
	if err != nil {
		return fmt.Errorf("could not read header: %w", err)
	}

	// ensure all checkpoint part file exists, might return os.ErrNotExist error
	// if a file is missing
	err = allPartFileExist(dir, fileName, len(subtrieChecksums))
	if err != nil {
		return fmt.Errorf("fail to check all checkpoint part file exist: %w", err)
	}

	err = readNodesFromSubTriesConcurrently(dir, fileName, result, subtrieChecksums, &lg)
	if err != nil {
		return fmt.Errorf("could not read subtrie from dir: %w", err)
	}

	return nil
}

type jobReadSubtrieLeaf struct {
	Index    int
	Checksum uint32
}

func readNodesFromSubTriesConcurrently(
	dir string, fileName string, result chan<- *ReadingResult, subtrieChecksums []uint32, logger *zerolog.Logger) error {

	numOfSubTries := len(subtrieChecksums)
	jobs := make(chan jobReadSubtrieLeaf, numOfSubTries)

	// push all jobs into the channel
	for i, checksum := range subtrieChecksums {
		jobs <- jobReadSubtrieLeaf{
			Index:    i,
			Checksum: checksum,
		}
	}
	close(jobs)

	nWorker := numOfSubTries // use as many worker as the jobs to read subtries concurrently
	for i := 0; i < nWorker; i++ {
		go func() {
			for job := range jobs {
				nodes, err := readLeafNodeFromCheckpointSubtrie(dir, fileName, job.Index, job.Checksum, logger)
				for _, leafNode := range nodes {
					//TODO does this work just as well?
					result <- &ReadingResult{
						LeafNode: leafNode,
						Err:      err,
					}
				}
			}
		}()
	}

	return nil
}

func readLeafNodeFromCheckpointSubtrie(dir string, fileName string, index int, checksum uint32, logger *zerolog.Logger) (leafNodes []*LeafNode, errToReturn error) {
	filePath, _, err := filePathSubTries(dir, fileName, index)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %v: %w", filePath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Err(err)
		}
	}(f)

	// validate the magic bytes and version
	err = validateFileHeader(wal.MagicBytesCheckpointSubtrie, wal.VersionV6, f)
	if err != nil {
		return nil, err
	}

	nodesCount, expectedSum, err := readSubTriesFooter(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read sub trie node count: %w", err)
	}

	if checksum != expectedSum {
		return nil, fmt.Errorf("mismatch checksum in subtrie file. checksum from checkpoint header %v does not "+
			"match with the checksum in subtrie file %v", checksum, expectedSum)
	}

	// restart from the beginning of the file, make sure CRC32Reader has seen all the bytes
	// in order to compute the correct checksum
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("cannot seek to start of file: %w", err)
	}

	reader := wal.NewCRC32Reader(bufio.NewReaderSize(f, defaultBufioReadSize))

	// read version again for calculating checksum
	_, _, err = readFileHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read version again for subtrie: %w", err)
	}

	// read file part index and verify
	scratch := make([]byte, 1024*4) // must not be less than 1024
	logging := logProgress(fmt.Sprintf("reading %v-th sub trie roots", index), int(nodesCount), logger)

	leafNodes = make([]*LeafNode, 0, nodesCount+1)
	nodes := make([]*node.Node, nodesCount+1)
	for i := uint64(1); i <= nodesCount; i++ {
		readNode, err := flattener.ReadNode(reader, scratch, func(nodeIndex uint64) (*node.Node, error) {
			if nodeIndex >= i {
				return nil, fmt.Errorf("sequence of serialized nodes does not satisfy Descendents-First-Relationship")
			}
			return nodes[i], nil
		})
		if err != nil {
			return nil, fmt.Errorf("cannot read readNode %d: %w", i, err)
		}
		if readNode.IsLeaf() {
			leafNodes = append(leafNodes,
				&LeafNode{
					Hash:    readNode.Hash(),
					Path:    *readNode.Path(),
					Payload: readNode.Payload(),
				})
		}
		logging(i)
	}

	// read footer and discard, since we only care about checksum
	_, err = io.ReadFull(reader, scratch[:encNodeCountSize])
	if err != nil {
		return nil, fmt.Errorf("cannot read footer: %w", err)
	}

	// calculate the actual checksum
	actualSum := reader.Crc32()

	if actualSum != expectedSum {
		return nil, fmt.Errorf("invalid checksum in subtrie checkpoint, expected %v, actual %v",
			expectedSum, actualSum)
	}

	// read the checksum and discard, since we only care about whether ensureReachedEOF
	_, err = io.ReadFull(reader, scratch[:crc32SumSize])
	if err != nil {
		return nil, fmt.Errorf("could not read subtrie file's checksum: %w", err)
	}

	err = ensureReachedEOF(reader)
	if err != nil {
		return nil, fmt.Errorf("fail to read %v-th sutrie file: %w", index, err)
	}

	return leafNodes, nil
}

// ensureReachedEOF checks if the reader has reached end of file
// it returns nil if reached EOF
// it returns ErrEOFNotReached if didn't reach end of file
// any error returned are exception
func ensureReachedEOF(reader io.Reader) error {
	b := make([]byte, 1)
	_, err := reader.Read(b)
	if errors.Is(err, io.EOF) {
		return nil
	}

	if err == nil {
		return ErrEOFNotReached
	}

	return fmt.Errorf("fail to check if reached EOF: %w", err)
}

// allPartFileExist check if all the part files of the checkpoint file exist
// it returns nil if all files exist
// it returns os.ErrNotExist if some file is missing, use (os.IsNotExist to check)
// it returns err if running into any exception
func allPartFileExist(dir string, fileName string, totalSubtrieFiles int) error {
	matched, err := findCheckpointPartFiles(dir, fileName)
	if err != nil {
		return fmt.Errorf("could not check all checkpoint part file exist: %w", err)
	}

	// header + subtrie files + top level file
	if len(matched) != 1+totalSubtrieFiles+1 {
		return fmt.Errorf("some checkpoint part file is missing. found part files %v. err :%w",
			matched, os.ErrNotExist)
	}

	return nil
}

func filePathPattern(dir string, fileName string) string {
	return fmt.Sprintf("%v*", filePathCheckpointHeader(dir, fileName))
}

func filePathCheckpointHeader(dir string, fileName string) string {
	return path.Join(dir, fileName)
}

func filePathSubTries(dir string, fileName string, index int) (string, string, error) {
	if index < 0 || index > (subtrieCount-1) {
		return "", "", fmt.Errorf("index must be between 0 to %v, but got %v", subtrieCount-1, index)
	}
	subTrieFileName := partFileName(fileName, index)
	return path.Join(dir, subTrieFileName), subTrieFileName, nil
}

func filePathTopTries(dir string, fileName string) (string, string) {
	topTriesFileName := partFileName(fileName, subtrieCount)
	return path.Join(dir, topTriesFileName), topTriesFileName
}

func partFileName(fileName string, index int) string {
	return fmt.Sprintf("%v.%03d", fileName, index)
}

// findCheckpointPartFiles returns a slice of file full paths of the part files for the checkpoint file
// with the given fileName under the given folder.
// - it return the matching part files, note it might not contains all the part files.
// - it return error if running any exception
func findCheckpointPartFiles(dir string, fileName string) ([]string, error) {
	pattern := filePathPattern(dir, fileName)
	matched, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("could not find checkpoint files: %w", err)
	}

	// build a lookup with matched
	lookup := make(map[string]struct{})
	for _, match := range matched {
		lookup[match] = struct{}{}
	}

	headerPath := filePathCheckpointHeader(dir, fileName)
	parts := make([]string, 0)
	// check header exists
	_, ok := lookup[headerPath]
	if ok {
		parts = append(parts, headerPath)
		delete(lookup, headerPath)
	}

	// check all subtrie parts
	for i := 0; i < subtrieCount; i++ {
		subtriePath, _, err := filePathSubTries(dir, fileName, i)
		if err != nil {
			return nil, err
		}
		_, ok := lookup[subtriePath]
		if ok {
			parts = append(parts, subtriePath)
			delete(lookup, subtriePath)
		}
	}

	// check top level trie part file
	toplevelPath, _ := filePathTopTries(dir, fileName)

	_, ok = lookup[toplevelPath]
	if ok {
		parts = append(parts, toplevelPath)
		delete(lookup, toplevelPath)
	}

	return parts, nil
}

func readCRC32Sum(reader io.Reader) (uint32, error) {
	bytes := make([]byte, crc32SumSize)
	_, err := io.ReadFull(reader, bytes)
	if err != nil {
		return 0, err
	}
	return decodeCRC32Sum(bytes)
}

func decodeCRC32Sum(encoded []byte) (uint32, error) {
	if len(encoded) != crc32SumSize {
		return 0, fmt.Errorf("wrong crc32sum size, expect %v, got %v", crc32SumSize, len(encoded))
	}
	return binary.BigEndian.Uint32(encoded), nil
}

func readSubtrieCount(reader io.Reader) (uint16, error) {
	bytes := make([]byte, encSubtrieCountSize)
	_, err := io.ReadFull(reader, bytes)
	if err != nil {
		return 0, err
	}
	return decodeSubtrieCount(bytes)
}

func decodeSubtrieCount(encoded []byte) (uint16, error) {
	if len(encoded) != encSubtrieCountSize {
		return 0, fmt.Errorf("wrong subtrie level size, expect %v, got %v", encSubtrieCountSize, len(encoded))
	}
	return binary.BigEndian.Uint16(encoded), nil
}

func validateFileHeader(expectedMagic uint16, expectedVersion uint16, reader io.Reader) error {
	magic, version, err := readFileHeader(reader)
	if err != nil {
		return err
	}

	if magic != expectedMagic {
		return fmt.Errorf("wrong magic bytes, expect %v, bot got: %v", expectedMagic, magic)
	}

	if version != expectedVersion {
		return fmt.Errorf("wrong version, expect %v, bot got: %v", expectedVersion, version)
	}

	return nil
}
func readFileHeader(reader io.Reader) (uint16, uint16, error) {
	bytes := make([]byte, headerSize)
	_, err := io.ReadFull(reader, bytes)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read magic ID and version: %w", err)
	}
	return decodeVersion(bytes)
}

func decodeVersion(encoded []byte) (uint16, uint16, error) {
	if len(encoded) != headerSize {
		return 0, 0, fmt.Errorf("wrong version size, expect %v, got %v", encMagicSize+encVersionSize, len(encoded))
	}
	magicBytes := binary.BigEndian.Uint16(encoded)
	version := binary.BigEndian.Uint16(encoded[encMagicSize:])
	return magicBytes, version, nil
}

func logProgress(msg string, estimatedSubtrieNodeCount int, logger *zerolog.Logger) func(nodeCounter uint64) {
	lookup := make(map[int]int)
	for i := 1; i < 10; i++ { // [1...9]
		lookup[estimatedSubtrieNodeCount/10*i] = i * 10
	}
	return func(nodeCounter uint64) {
		percentage, ok := lookup[int(nodeCounter)]
		if ok {
			logger.Info().Msgf("%s completion percentage: %v percent", msg, percentage)
		}
	}
}

func decodeNodeCount(encoded []byte) (uint64, error) {
	if len(encoded) != encNodeCountSize {
		return 0, fmt.Errorf("wrong subtrie node count size, expect %v, got %v", encNodeCountSize, len(encoded))
	}
	return binary.BigEndian.Uint64(encoded), nil
}

func readSubTriesFooter(f *os.File) (uint64, uint32, error) {
	const footerSize = encNodeCountSize // footer doesn't include crc32 sum
	const footerOffset = footerSize + crc32SumSize
	_, err := f.Seek(-footerOffset, io.SeekEnd)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot seek to footer: %w", err)
	}

	footer := make([]byte, footerSize)
	_, err = io.ReadFull(f, footer)
	if err != nil {
		return 0, 0, fmt.Errorf("could not read footer: %w", err)
	}

	nodeCount, err := decodeNodeCount(footer)
	if err != nil {
		return 0, 0, fmt.Errorf("could not decode subtrie node count: %w", err)
	}

	// the subtrie checksum from the checkpoint header file must be same
	// as the checksum included in the subtrie file
	expectedSum, err := readCRC32Sum(f)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read checksum for sub trie file: %w", err)
	}

	return nodeCount, expectedSum, nil
}
