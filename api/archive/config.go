package archive

import (
	"github.com/onflow/flow-archive/service/trace"
)

var DefaultConfig = Config{
	tracer: trace.NewNoopTracer(),
}

type Config struct {
	tracer trace.Tracer
}

type Option func(*Config)

func WithTracer(tracer trace.Tracer) Option {
	return func(cfg *Config) {
		cfg.tracer = tracer
	}
}
