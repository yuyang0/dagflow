package flow

import (
	"crypto/rand"
	"math/big"
)

// Aggregator definition for the data aggregator of nodes
type Aggregator func(map[string][]byte) ([]byte, error)

// Forwarder definition for the data forwarder of nodes
type Forwarder func([]byte) []byte

// ForEach definition for the foreach function
type ForEach func([]byte) map[string][]byte

type FuncErrorHandler func(error) error

type ExecutionOptions struct {
	aggregator Aggregator
	// forwarder      Forwarder
	// noForwarder    bool
	failureHandler      FuncErrorHandler
	finalFailureHandler FuncErrorHandler
}

type Option func(*ExecutionOptions)

func WithAggregator(agg Aggregator) Option {
	return func(o *ExecutionOptions) {
		o.aggregator = agg
	}
}

func WithFailureHandler(fn FuncErrorHandler) Option {
	return func(o *ExecutionOptions) {
		o.failureHandler = fn
	}
}

func WithFinalFailureHandler(fn FuncErrorHandler) Option {
	return func(o *ExecutionOptions) {
		o.finalFailureHandler = fn
	}
}

func randomAgg(dataMap map[string][]byte) ([]byte, error) {
	bg := big.NewInt(int64(len(dataMap)))
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		return nil, err
	}
	k := n.Int64()
	i := int64(0)

	for _, data := range dataMap {
		if i == k {
			return data, nil
		}
		i++
	}
	return nil, nil
}
