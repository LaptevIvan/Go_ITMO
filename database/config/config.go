package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultAttemptsRetry = 2000
	defaultLogValue      = true
	defaultMaxConn       = "10"
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

		Outbox struct {
			Enabled         bool          `env:"OUTBOX_ENABLED"`
			Workers         int           `env:"OUTBOX_WORKERS"`
			BatchSize       int           `env:"OUTBOX_BATCH_SIZE"`
			WaitTimeMS      time.Duration `env:"OUTBOX_WAIT_TIME_MS"`
			InProgressTTLMS time.Duration `env:"OUTBOX_IN_PROGRESS_TTL_MS"`
			AuthorSendURL   string        `env:"OUTBOX_AUTHOR_SEND_URL"`
			BookSendURL     string        `env:"OUTBOX_BOOK_SEND_URL"`
			AttemptsRetry   int           `env:"OUTBOX_ATTEMPTS_RETRY"`
		}

		Log struct {
			LogController   bool `env:"LOG_CONTROLLER_ENABLED"`
			LogTransactor   bool `env:"LOG_TRANSACTOR_ENABLED"`
			LogUseCase      bool `env:"LOG_USECASE_ENABLED"`
			LogDBRepo       bool `env:"LOG_DB_REPO_ENABLED"`
			LogOutboxWorker bool `env:"LOG_OUTBOX_WORKER_ENABLED"`
		}

		Observability struct {
			MetricsPort string `env:"METRICS_PORT"`
			JaegerURL   string `env:"JAEGER_URL"`
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

	var err error
	v := viper.New()
	if cfg.PG.MaxConn, err = parseEnvString(v, "db_MaxCon", "POSTGRES_MAX_CONN", defaultMaxConn); err != nil {
		return nil, err
	}

	cfg.PG.URL = fmt.Sprintf("postgres://%s:%s@", cfg.PG.User, cfg.PG.Password) +
		net.JoinHostPort(cfg.PG.Host, cfg.PG.Port) + fmt.Sprintf("/%s?sslmode=disable", cfg.PG.DB) + fmt.Sprintf("&pool_max_conns=%s", cfg.PG.MaxConn)

	if cfg.Outbox.Enabled, err = parseEnvBool(v, "outbox", "OUTBOX_ENABLED"); err != nil {
		return nil, err
	}

	if cfg.Outbox.Enabled {
		if cfg.Outbox.Workers, err = parseInt(os.Getenv("OUTBOX_WORKERS")); err != nil {
			return nil, err
		}

		if cfg.Outbox.BatchSize, err = parseInt(os.Getenv("OUTBOX_BATCH_SIZE")); err != nil {
			return nil, err
		}

		if cfg.Outbox.WaitTimeMS, err = parseTime(os.Getenv("OUTBOX_WAIT_TIME_MS")); err != nil {
			return nil, err
		}

		if cfg.Outbox.InProgressTTLMS, err = parseTime(os.Getenv("OUTBOX_IN_PROGRESS_TTL_MS")); err != nil {
			return nil, err
		}

		cfg.Outbox.AuthorSendURL = os.Getenv("OUTBOX_AUTHOR_SEND_URL")
		cfg.Outbox.BookSendURL = os.Getenv("OUTBOX_BOOK_SEND_URL")

		if cfg.Outbox.AttemptsRetry, err = parseEnvInt(v, "attempts", "OUTBOX_ATTEMPTS_RETRY", defaultAttemptsRetry); err != nil {
			return nil, err
		}
	}

	if cfg.Log.LogController, err = parseEnvBool(v, "log_controller", "LOG_CONTROLLER_ENABLED", defaultLogValue); err != nil {
		return nil, err
	}

	if cfg.Log.LogTransactor, err = parseEnvBool(v, "log_transactor", "LOG_TRANSACTOR_ENABLED", defaultLogValue); err != nil {
		return nil, err
	}

	if cfg.Log.LogUseCase, err = parseEnvBool(v, "log_usecase", "LOG_USECASE_ENABLED", defaultLogValue); err != nil {
		return nil, err
	}

	if cfg.Log.LogDBRepo, err = parseEnvBool(v, "log_db", "LOG_DB_REPO_ENABLED", defaultLogValue); err != nil {
		return nil, err
	}

	if cfg.Log.LogOutboxWorker, err = parseEnvBool(v, "log_outbox_worker", "LOG_OUTBOX_WORKER_ENABLED", defaultLogValue); err != nil {
		return nil, err
	}

	cfg.Observability.MetricsPort = os.Getenv("METRICS_PORT")
	cfg.Observability.JaegerURL = os.Getenv("JAEGER_URL")

	return cfg, nil
}

func parseTime(s string) (time.Duration, error) {
	t, err := parseInt(s)

	if err != nil {
		return time.Duration(0), err
	}

	return time.Duration(t) * time.Millisecond, nil
}

func parseInt(s string) (int, error) {
	str, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return 0, err
	}

	return int(str), nil
}

func parseEnvBool(v *viper.Viper, key, envVar string, defaultValue ...bool) (bool, error) {
	err := v.BindEnv(key, envVar)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0], err
		}
		return false, err
	}
	if len(defaultValue) > 0 {
		v.SetDefault(key, defaultValue[0])
	}
	return v.GetBool(key), nil
}

func parseEnvInt(v *viper.Viper, key, envVar string, defaultValue ...int) (int, error) {
	err := v.BindEnv(key, envVar)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0], err
		}
		return 0, err
	}
	if len(defaultValue) > 0 {
		v.SetDefault(key, defaultValue[0])
	}
	return v.GetInt(key), nil
}

func parseEnvString(v *viper.Viper, key, envVar string, defaultValue ...string) (string, error) {
	err := v.BindEnv(key, envVar)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0], err
		}
		return "", err
	}
	if len(defaultValue) > 0 {
		v.SetDefault(key, defaultValue[0])
	}
	return v.GetString(key), nil
}
