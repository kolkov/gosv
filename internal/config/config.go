package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Processes []ProcessConfig `yaml:"processes"`
}

type ProcessConfig struct {
	Name        string            `yaml:"name"`
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args,omitempty"`
	Directory   string            `yaml:"directory,omitempty"`
	Environment map[string]string `yaml:"env,omitempty"`
	Autostart   bool              `yaml:"autostart"`
	Autorestart string            `yaml:"autorestart"`
	StopSignal  string            `yaml:"stop_signal,omitempty"`
	StopWait    time.Duration     `yaml:"stop_wait,omitempty"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	for i := range cfg.Processes {
		if cfg.Processes[i].Directory != "" {
			if abs, err := filepath.Abs(cfg.Processes[i].Directory); err == nil {
				cfg.Processes[i].Directory = abs
			}
		}

		if cfg.Processes[i].StopSignal == "" {
			cfg.Processes[i].StopSignal = "SIGTERM"
		}

		if cfg.Processes[i].StopWait == 0 {
			cfg.Processes[i].StopWait = 10 * time.Second
		}
	}

	return &cfg, nil
}
