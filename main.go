package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Server    ServerConfig    `json:"server"`
	Converter ConverterConfig `json:"converter"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ConverterConfig struct {
	OutputDir string `json:"output_dir"`
}

func main() {
	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Println("Failed to load config.json:", err)
		return
	}

	addr := fmt.Sprintf(
		"%s:%d",
		cfg.Server.Host,
		cfg.Server.Port,
	)

	log.Printf("Server started at http://%s\n", addr)
	StartServer(addr, cfg.Converter.OutputDir)
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
