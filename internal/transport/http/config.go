package http

import "time"

type Config struct {
	HTTP struct {
		Port         int           `yaml:"port"          env:"SERVER_HTTP_PORT"`
		ReadTimeout  time.Duration `yaml:"read_timeout"  env:"SERVER_HTTP_READ_TIMEOUT"`
		WriteTimeout time.Duration `yaml:"write_timeout" env:"SERVER_HTTP_WRITE_TIMEOUT"`

		BasicAuth struct {
			Enabled  bool   `yaml:"enabled"   env:"SERVER_HTTP_BASICAUTH_ENABLED"`
			Username string `yaml:"username"  env:"SERVER_HTTP_BASICAUTH_USERNAME"`
			Password string `env:"SERVER_HTTP_BASICAUTH_PASSWORD"`
		} `yaml:"basic_auth"`
	} `yaml:"http"`

	GracefulShutdown time.Duration `yaml:"graceful_shutdown"  env:"SERVER_GRACEFUL_SHUTDOWN"`
}
