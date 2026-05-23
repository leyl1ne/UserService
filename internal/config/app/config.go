package app

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/password"
	"github.com/leyl1ne/UserService/internal/infrastructure/postgres"
	"github.com/leyl1ne/UserService/internal/logger"
	httpServer "github.com/leyl1ne/UserService/internal/transport/http"
)

type Config struct {
	App            AppConfig         `yaml:"app"`
	Server         httpServer.Config `yaml:"server"`
	Logger         logger.Config     `yaml:"logger"`
	Postgres       postgres.Config   `yaml:"postgres"`
	PasswordHasher password.Config   `yaml:"password_hasher"`
	JWTGenerator   jwt.Config        `yaml:"jwt_generator"`
}

type AppConfig struct {
	Name        string `yaml:"name"        env:"APP_NAME"`
	Version     string `yaml:"version"     env:"APP_VERSION"`
	Environment string `yaml:"environment" env:"APP_ENV" env-default:"local"`
}

func LoadConfig() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		return nil, fmt.Errorf("CONFIG_PATH environment variable is not set")
	}

	// Проверяем существование конфиг-файла
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}

	var cfg Config

	// Читаем конфиг-файл и заполняем нашу структуру
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return &cfg, nil
}
