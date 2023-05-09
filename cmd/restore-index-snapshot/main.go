package main

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-archive/codec/zbor"
	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/service/index"
	"github.com/onflow/flow-archive/service/storage"
	"github.com/onflow/flow-archive/service/storage2"
)

const (
	success = 0
	failure = 1
)

const (
	encodingNone   = "none"
	encodingHex    = "hex"
	encodingBase64 = "base64"
)

const (
	compressionNone = "none"
	compressionZstd = "zstd"
	compressionGzip = "gzip"
)

func main() {
	os.Exit(run())
}

func run() int {

	// Parse the command line arguments.
	var (
		flagCompression string
		flagEncoding    string

		flagIndex          string
		flagIndex2         string
		flagBlockCacheSize int64
	)

	pflag.StringVarP(&flagCompression, "compression", "c", compressionZstd, "compression algorithm (\"none\", \"zstd\" or \"gzip\")")
	pflag.StringVarP(&flagEncoding, "encoding", "e", encodingNone, "output encoding (\"none\", \"hex\" or \"base64\")")

	pflag.StringVarP(&flagIndex, "index", "i", "index", "path to database directory for state index")
	pflag.StringVarP(&flagIndex2, "index2", "I", "index2", "path to the pebble-based index database directory")
	pflag.Int64Var(&flagBlockCacheSize, "block-cache-size", 1<<30, "size of the pebble block cache in bytes.")

	pflag.Parse()

	// Initialize the logger.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)

	// Open the index database.
	db, err := badger.Open(archive.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open badger db")
		return failure
	}
	defer db.Close()

	storage2, err := storage2.NewLibrary2(flagIndex2, flagBlockCacheSize)
	if err != nil {
		log.Error().Str("index2", flagIndex2).Err(err).Msg("could not open storage2")
		return failure
	}
	defer func() {
		err := storage2.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close storage2")
		}
	}()

	// Check if the database is empty.
	index := index.NewReader(log, db, storage.New(zbor.NewCodec()), storage2)
	_, err = index.First()
	if err == nil {
		log.Error().Msg("database directory already contains index database")
		return failure
	}

	// We will consume from stdin; if the user wants to load from a file, he can
	// pipe it into the command.
	var reader io.Reader
	reader = os.Stdin
	defer os.Stdin.Close()

	// When reading, we first need to decompress, so we start with that
	switch flagCompression {
	case compressionNone:
		// nothing to do
	case compressionZstd:
		decompressor, _ := zstd.NewReader(reader)
		defer decompressor.Close()
		reader = decompressor
	case compressionGzip:
		decompressor, _ := gzip.NewReader(reader)
		defer decompressor.Close()
		reader = decompressor
	default:
		log.Error().Str("compression", flagCompression).Msg("invalid compression algorithm specified")
	}

	// After decompression, we can decode the encoding.
	switch flagEncoding {
	case encodingNone:
		// nothing to do
	case encodingHex:
		reader = hex.NewDecoder(reader)
	case encodingBase64:
		reader = base64.NewDecoder(base64.StdEncoding, reader)
	default:
		log.Error().Str("encoding", flagEncoding).Msg("invalid encoding format specified")
	}

	// Restore the database
	err = db.Load(reader, runtime.GOMAXPROCS(0))
	if err != nil {
		log.Error().Err(err).Msg("snapshot restoration failed")
		return failure
	}

	log.Info().Msg("snapshot restoration complete")

	return success
}
