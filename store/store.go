package store

import "context"

type Store interface {
	Set(ctx context.Context, k string, v []byte) error
	// Get returns the value associated with the key.
	// please note that Get returns (nil, nil) if the key does not exist.
	Get(ctx context.Context, k string) ([]byte, error)
}
