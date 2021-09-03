module github.com/optakt/flow-dps

go 1.16

require (
	cloud.google.com/go/storage v1.16.0
	github.com/OneOfOne/xxhash v1.2.8
	github.com/c2h5oh/datasize v0.0.0-20200825124411-48ed595a09d2
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/dgraph-io/ristretto v0.0.3-0.20200630154024-f66de99634de
	github.com/envoyproxy/protoc-gen-validate v0.6.1
	github.com/fxamacker/cbor/v2 v2.2.1-0.20210510192846-c3f3c69e7bc8
	github.com/gammazero/deque v0.1.0
	github.com/go-playground/validator/v10 v10.9.0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware/providers/zerolog/v2 v2.0.0-rc.2
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.0-20200501113911-9a95f0fdbfea
	github.com/hashicorp/go-multierror v1.1.1
	github.com/klauspost/compress v1.12.2
	github.com/labstack/echo/v4 v4.3.0
	github.com/onflow/cadence v0.18.1-0.20210730161646-b891a21c51fd
	github.com/onflow/flow-go v0.21.1-0.20210902230851-cd710a1464fc
	github.com/onflow/flow-go-sdk v0.21.0
	github.com/onflow/flow-go/crypto v0.18.0
	github.com/onflow/flow/protobuf/go/flow v0.2.2
	github.com/prometheus/tsdb v0.7.1
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/rs/zerolog v1.22.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/ziflex/lecho/v2 v2.3.1
	golang.org/x/mod v0.4.2
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/api v0.49.0
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/onflow/flow-go/crypto => ./flow-go/crypto
