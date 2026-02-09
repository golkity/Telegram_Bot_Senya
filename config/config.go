package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig() *Config {
	v := viper.New()
	v.AddConfigPath("config")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Println("Config file not found, using .env variables")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Config unmarshal failed: %v", err)
	}

	return &cfg
}
