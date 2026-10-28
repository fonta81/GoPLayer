package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fonta81/Reproductor-go/internal/player/eq"
)

type Config struct {
	MusicDir  string    `json:"music_dir"`
	EQEnabled bool      `json:"eq_enabled"`
	EQGains   []float64 `json:"eq_gains,omitempty"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	cfgDir := filepath.Join(dir, "goplayer")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "config.json"), nil
}

func loadConfig() (Config, error) {
	var c Config
	p, err := configPath()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

func saveConfig(c Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func LoadMusicDir() (string, error) {
	c, err := loadConfig()
	if err != nil {
		return "", err
	}
	return c.MusicDir, nil
}

func LoadEQ() (bool, [eq.BandCount]float64, error) {
	var gains [eq.BandCount]float64
	c, err := loadConfig()
	if err != nil {
		return false, gains, err
	}
	if len(c.EQGains) == 0 {
		return true, gains, nil
	}
	copy(gains[:], c.EQGains)
	return c.EQEnabled, gains, nil
}

func SaveMusicDir(dir string) error {
	c, _ := loadConfig()
	c.MusicDir = dir
	return saveConfig(c)
}

func SaveEQ(enabled bool, gains [eq.BandCount]float64) error {
	c, _ := loadConfig()
	c.EQEnabled = enabled
	c.EQGains = gains[:]
	return saveConfig(c)
}
