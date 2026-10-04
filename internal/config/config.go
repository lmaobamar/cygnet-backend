package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        uint16
	Storage     StorageConfig
}

var (
	once sync.Once
	cfg  Config
)

func Load(path ...string) Config {
	loadStart := time.Now()
	file := "config.json"
	if len(path) > 0 && path[0] != "" {
		file = path[0]
	}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("can't read %s: %v", path, err)
	}

	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		panic(fmt.Sprintf("can't parse %s: %v", file, err))
	}

	if cfg.DatabaseURL == "" {
		panic("DatabaseURL unset")
	}
	if cfg.Port == 0 {
		panic("Port is required")
	}
	if err := cfg.Storage.Validate(); err != nil {
		panic(fmt.Sprintf("can't load StorageConfig: %v", err))
	}

	log.Printf("cfg in %v", time.Since(loadStart))
	return cfg
}

func Get() Config {
	once.Do(func() { cfg = Load() })
	return cfg
}
