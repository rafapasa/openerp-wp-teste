package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"rafapasa/openerp-wp-teste/internal/dto"
	"sort"
)

func Load(path string) (*dto.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c dto.Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if c.Threads <= 0 {
		c.Threads = 5
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 60
	}
	if c.PollIntervalMs <= 0 {
		c.PollIntervalMs = 500
	}
	if c.ScenariosDir == "" {
		c.ScenariosDir = "./scenarios"
	}
	return &c, nil
}

func LoadScenarios(dir string) ([]dto.Scenario, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []dto.Scenario
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var s dto.Scenario
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
