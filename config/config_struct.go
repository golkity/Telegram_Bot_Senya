package config

type Config struct {
	Env      string   `mapstructure:"env"`
	Postgres Postgres `mapstructure:"postgres"`
	Redis    Redis    `mapstructure:"redis"`
	MinIO    MinIO    `mapstructure:"minio"`
	Telegram Telegram `mapstructure:"telegram"`
	Security Security `mapstructure:"security"`
}

type Security struct {
	EncryptionKey string `mapstructure:"encryption_key"`
	PathSalt      string `mapstructure:"path_salt"`
}

type Postgres struct {
	URL     string `mapstructure:"url"`
	PoolMax int    `mapstructure:"pool_max"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type MinIO struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type Telegram struct {
	Token  string  `mapstructure:"token"`
	Admins []int64 `mapstructure:"admins"`
}
