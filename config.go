package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func saveConfig(pathRelativeToHome string, c Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding HOME: %s", err)
	}

	// O_TRUNC allows us to re-write the whole configuration file every time we
	// save it. Without it, the file would never shrink in size when writing a
	// shorter configuration manifest, and the YAML manifest would break.
	f, err := os.OpenFile(home+"/"+pathRelativeToHome, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening config '%s': %s", pathRelativeToHome, err)
	}
	defer f.Close()

	err = yaml.NewEncoder(f).Encode(&c)
	if err != nil {
		return fmt.Errorf("encoding into YAML: %s", err)
	}

	return nil
}

type Config struct {
	Token     string `yaml:"token"`
	Workspace string `yaml:"workspace,omitempty"`
}

func loadConfig(pathRelativeToHome string) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("finding HOME: %s", err)
	}

	path := home + "/" + pathRelativeToHome

	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	_, isPathErr := err.(*os.PathError)
	switch {
	case isPathErr:
		// Probably a 'not found' error, we don't need to worry the user
		// when the configuration file does not exist yet.
		return Config{}, nil
	case err != nil:
		return Config{}, fmt.Errorf("opening config '%s': %s", path, err)
	}
	defer f.Close()

	c := Config{}
	err = yaml.NewDecoder(f).Decode(&c)
	if err != nil {
		return Config{}, fmt.Errorf("decoding '%s' from YAML: %s", path, err)
	}

	return c, nil
}
