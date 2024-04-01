package types

type Config struct {
	Store StoreConfig `json:"store"`
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
