package types

import (
	"errors"
	"time"

	"github.com/mcuadros/go-defaults"
)

type Config struct {
	Store StoreConfig `json:"store"`
	// some configs related to asynq
	Redis         RedisConfig   `json:"redis"`
	UseAsynqStore bool          `json:"useAsynqStore" default:"true"`
	RetryCount    int           `json:"retryCount" default:"30"`
	Timeout       time.Duration `json:"timeout" default:"5h"`
}

type StoreConfig struct {
	Type  string      `json:"type" default:"redis"`
	Redis RedisConfig `json:"redis"`
}

type RedisConfig struct {
	Addr          string   `json:"addr"`
	SentinelAddrs []string `json:"sentinel_addrs"`
	MasterName    string   `json:"master_name"`
	Username      string   `json:"username"`
	Password      string   `json:"password"`
	DB            int      `json:"db"`
	Expire        uint     `json:"expire"`
}

func (rCfg *RedisConfig) check() error {
	if rCfg.Addr == "" && len(rCfg.SentinelAddrs) == 0 {
		return errors.New("add or snetinel addrs should be specified")
	}
	return nil
}
func (cfg *Config) Refine() error {
	if err := cfg.Redis.check(); err != nil {
		return err
	}
	if cfg.Store.Redis.Addr == "" && len(cfg.Store.Redis.SentinelAddrs) == 0 {
		cfg.Store.Redis = cfg.Redis
	}
	defaults.SetDefaults(cfg)
	return nil
}
