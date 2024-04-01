package flow

import "math/rand"

// Aggregator definition for the data aggregator of nodes
type Aggregator func(map[string][]byte) ([]byte, error)

// Forwarder definition for the data forwarder of nodes
type Forwarder func([]byte) []byte

// ForEach definition for the foreach function
type ForEach func([]byte) map[string][]byte

type FuncErrorHandler func(error) error

type ExecutionOptions struct {
	aggregator     Aggregator
	forwarder      Forwarder
	noForwarder    bool
	failureHandler FuncErrorHandler
}

type Option func(*ExecutionOptions)

func WithAggregator(agg Aggregator) Option {
	return func(o *ExecutionOptions) {
		o.aggregator = agg
	}
}

func randomAgg(dataMap map[string][]byte) ([]byte, error) {
	k := rand.Intn(len(dataMap))
	i := 0

	for _, data := range dataMap {
		if i == k {
			return data, nil
		}
		i++
	}
	return nil, nil
}
