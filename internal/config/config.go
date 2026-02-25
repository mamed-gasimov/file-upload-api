package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`

	Postgres struct {
		Host     string `env:"HOST" envDefault:"localhost"`
		Port     string `env:"PORT" envDefault:"5432"`
		User     string `env:"USER,required" envDefault:"fileuser"`
		Password string `env:"PASSWORD,required" envDefault:"filepass"`
		Database string `env:"DB,required" envDefault:"filedb"`
		SSLMode  string `env:"SSL" envDefault:"false"`
	} `envPrefix:"POSTGRES_"`

	Minio struct {
		Endpoint  string `env:"ENDPOINT" envDefault:"http://localhost:9000"`
		AccessKey string `env:"ACCESS_KEY,required" envDefault:"minioadmin"`
		SecretKey string `env:"SECRET_KEY,required" envDefault:"minioadmin"`
		Bucket    string `env:"BUCKET,required" envDefault:"files"`
		UseSSL    bool   `env:"SSL" envDefault:"false"`
	} `envPrefix:"MINIO_"`

	OpenAI struct {
		APIKey  string `env:"API_KEY,required" envDefault:""`
		BaseURL string `env:"BASE_URL,required" envDefault:"https://api.openai.com/v1/"`
	} `envPrefix:"OPENAI_"`
}

func Load() (*Config, error) {
	if _, err := os.Stat("./.env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Postgres.User, c.Postgres.Password,
		c.Postgres.Host, c.Postgres.Port,
		c.Postgres.Database, c.Postgres.SSLMode,
	)
}
