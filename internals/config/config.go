package config

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Database struct {
		SQLitePath string `yaml:"sqlite_path"`
	} `yaml:"database"`

	Kafka struct {
		Brokers []string `yaml:"brokers"`
		Topic   string   `yaml:"topic"`
	} `yaml:"kafka"`

	Game struct {
		MatchmakingTimeoutSeconds int `yaml:"matchmaking_timeout_seconds"`
		ReconnectTimeoutSeconds   int `yaml:"reconnect_timeout_seconds"`
		BoardRows                 int `yaml:"board_rows"`
		BoardColumns              int `yaml:"board_columns"`
	} `yaml:"game"`
}

func MustLoad() *Config {
	var configPath string
	configPath = os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configflag := flag.String("config", "", "Path to configuration file")
		flag.Parse()
		configPath = *configflag
		if configPath == "" {
			log.Fatal("Config Path is not set")
		}
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Config file does not exist: %s", configPath)
	}
	var cfg Config
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	return &cfg
}
