package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

const DefaultMaxConcurrency = 2
const DefaultScanInterval = "5m"

type Config struct {
	Config configSection
	Sync   []syncSection
}

type configSection struct {
	Token             string
	MaxConcurrency    int
	MaxBytesPerSecond int
	ScanInterval      duration
}

type syncSection struct {
	Remote string
	Local  string
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func LoadConfigFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return LoadConfigFromReader(f)
}

func LoadConfigFromReader(r io.Reader) (*Config, error) {
	var c Config
	meta, err := toml.DecodeReader(r, &c)
	if err != nil {
		return nil, err
	}
	if !meta.IsDefined("config", "maxConcurrency") {
		c.Config.MaxConcurrency = DefaultMaxConcurrency
	}
	if !meta.IsDefined("config", "scanInterval") {
		dur, _ := time.ParseDuration(DefaultScanInterval)
		c.Config.ScanInterval.Duration = dur
	}
	for i, s := range c.Sync {
		if s.Local == "" {
			return nil, fmt.Errorf("Sync entry %d: local path cannot be empty", i)
		}
	}
	return &c, nil
}
