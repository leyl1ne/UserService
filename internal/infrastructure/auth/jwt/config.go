package jwt

import "time"

type Config struct {
	Secret string        `yaml:"secret"`
	TTL    time.Duration `yaml:"ttl"`
}
