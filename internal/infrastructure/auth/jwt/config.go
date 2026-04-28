package jwt

import "time"

type Config struct {
	Secret        string        `yaml:"secret"`
	AccessTokeTTL time.Duration `yaml:"access_token_ttl"`
}
