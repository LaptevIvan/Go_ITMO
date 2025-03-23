package config

import (
	"fmt"
	"net"
	"os"
)

type (
	Config struct {
		GRPC struct {
			Port        string `env:"GRPC_PORT"`
			GatewayPort string `env:"GRPC_GATEWAY_PORT"`
		}

		PG struct {
			URL      string
			Host     string `env:"POSTGRES_HOST"`
			Port     string `env:"POSTGRES_PORT"`
			DB       string `env:"POSTGRES_DB"`
			User     string `env:"POSTGRES_USER"`
			Password string `env:"POSTGRES_PASSWORD"`
			MaxConn  string `env:"POSTGRES_MAX_CONN"`
		}
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	cfg.GRPC.Port = os.Getenv("GRPC_PORT")
	cfg.GRPC.GatewayPort = os.Getenv("GRPC_GATEWAY_PORT")

	cfg.PG.Host = os.Getenv("POSTGRES_HOST")
	cfg.PG.Port = os.Getenv("POSTGRES_PORT")
	cfg.PG.DB = os.Getenv("POSTGRES_DB")
	cfg.PG.User = os.Getenv("POSTGRES_USER")
	cfg.PG.Password = os.Getenv("POSTGRES_PASSWORD")
	cfg.PG.MaxConn = os.Getenv("POSTGRES_MAX_CONN")

	cfg.PG.URL = fmt.Sprintf("postgres://%s:%s@", cfg.PG.User, cfg.PG.Password) +
		net.JoinHostPort(cfg.PG.Host, cfg.PG.Port) + fmt.Sprintf("/%s?sslmode=disable", cfg.PG.DB)

	return cfg, nil
}
