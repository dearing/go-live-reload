package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Builds      []Build `json:"builds"`
}

// Save saves a json representation of Config to filename
//
//	ex: myConfig.Save("go-live-reload.json")
func (c *Config) Save(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Load reads filename into a Config struct
//
//	ex: myConfig.Load("go-live-reload.json")
func (c *Config) Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, c)
	if err != nil {
		return err
	}
	return nil
}
