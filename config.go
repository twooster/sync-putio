package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

/*
[config]
token =
maxConcurrency = 2
scanInterval = "10m"

[[sync]]
remote = "movies"
local = "/var/downloads"
*/

const DEFAULT_MAX_CONCURRENCY = 2
const DEFAULT_SCAN_INTERVAL = "5m"

type Config struct {
	Config configSection
	Sync   []syncSection
}

type configSection struct {
	Token          string
	MaxConcurrency int
	ScanInterval   duration
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
		c.Config.MaxConcurrency = DEFAULT_MAX_CONCURRENCY
	}
	if !meta.IsDefined("config", "scanInterval") {
		dur, _ := time.ParseDuration(DEFAULT_SCAN_INTERVAL)
		c.Config.ScanInterval.Duration = dur
	}
	for i, s := range c.Sync {
		if s.Local == "" {
			return nil, fmt.Errorf("Sync entry %d: local directory cannot be empty", i)
		}
	}
	return &c, nil
}
