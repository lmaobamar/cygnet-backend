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
	JWTSecret   string
	Port        uint16
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
	if len(cfg.JWTSecret) < 32 {
		panic("JWTSecret must be at least 32 characters")
	}
	if cfg.Port == 0 {
		panic("Port is required")
	}

	log.Printf("cfg in %v", time.Since(loadStart))
	return cfg
}

func Get() Config {
	once.Do(func() { cfg = Load() })
	return cfg
}
