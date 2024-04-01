package factory

import (
	"github.com/cockroachdb/errors"
	"github.com/yuyang0/dagflow/store"
	"github.com/yuyang0/dagflow/store/redis"
	"github.com/yuyang0/dagflow/types"
)

func NewStore(cfg *types.StoreConfig) (store.Store, error) {
	switch cfg.Type {
	case "redis":
		return redis.New(&cfg.Redis)
	default:
		return nil, errors.Newf("invalid store type %s", cfg.Type)
	}
}
