package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yuyang0/dagflow/types"
)

const (
	prefix = "_dagflow"
)

type Store struct {
	cli *redis.Client
	cfg *types.RedisConfig
}

func New(cfg *types.RedisConfig) (*Store, error) {
	var cli *redis.Client
	if len(cfg.SentinelAddrs) > 0 {
		cli = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			DB:            cfg.DB,
			Username:      cfg.Username,
			Password:      cfg.Password,
		})
	} else {
		cli = redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			DB:       cfg.DB,
			Username: cfg.Username,
			Password: cfg.Password,
		})
	}
	return &Store{
		cli: cli,
		cfg: cfg,
	}, nil
}

func (s *Store) Set(ctx context.Context, k string, v []byte) error {
	key := fmt.Sprintf("%s_%s", prefix, k)
	return s.cli.Set(ctx, key, v, time.Duration(s.cfg.Expire)*time.Second).Err()
}

func (s *Store) Get(ctx context.Context, k string) ([]byte, error) {
	key := fmt.Sprintf("%s_%s", prefix, k)
	obj, err := s.cli.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return []byte(obj), nil
}
