package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Addr      string `env:"ADDR"       envDefault:":8080"`
	MongoURI  string `env:"MONGO_URI"  envDefault:"mongodb://localhost:27017"`
	MongoDB   string `env:"MONGO_DB"   envDefault:"taskdb"`
	JWTSecret string `env:"JWT_SECRET" envDefault:"secret"`
}

// Load reads environment variables (and an optional .env file) into Config.
func Load() (*Config, error) {
	_ = godotenv.Load() // silently ignore if .env is absent
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
