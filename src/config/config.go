package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port          string
	DBPath        string
	DeepSeekKey   string
	DeepSeekURL   string
	DeepSeekModel string
	BaseDir       string
	StaticDir     string
	DataDir       string
	LogDir        string
}

func Load() *Config {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)

	cfg := &Config{
		Port:          "41400",
		DeepSeekKey:   "",
		DeepSeekURL:   "https://api.deepseek.com",
		DeepSeekModel: "deepseek-v4-flash",
		BaseDir:       baseDir,
		StaticDir:     filepath.Join(baseDir, "static"),
		DataDir:       filepath.Join(baseDir, "data"),
		LogDir:        filepath.Join(baseDir, "data", "log"),
		DBPath:        filepath.Join(baseDir, "data", "assdata.db"),
	}
	return cfg
}
