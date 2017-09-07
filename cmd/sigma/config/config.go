package config

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	"github.com/homebot/sigma/launcher/docker"

	yaml "gopkg.in/yaml.v2"
)

// SigmaServerConfig is the configuration for the sigma server
type SigmaServerConfig struct {
	// Listen holds the address the sigma server should listen on
	Listen string `json:"listen" yaml:"listen"`
}

// NodeServerConfig is the configuration for the node handler server
type NodeServerConfig struct {
	// Listen holds the address the node handler server should listen on
	Listen string `json:"listen" yaml:"listen"`

	// AdvertiseAddress holds the address to advertise to new node
	// instances
	AdvertiseAddress string `json:"advertise" yaml:"advertise"`
}

// ProcessTypeConfig holds type configuration values for a process launcher
type ProcessTypeConfig struct {
	// Command holds the command to execute for the exec type
	Command []string `json:"command" yaml:"command"`
}

// ProcessLauncherConfig is the configuration for a process launcher
type ProcessLauncherConfig struct {
	// Types holds types supported by the launcher
	Types map[string]ProcessTypeConfig `json:"types" yaml:"types"`
}

// Launcher is the configuration for a launcher
type Launcher struct {
	// Docker is the configuration for the docker launcher
	Docker *docker.Config `json:"docker" yaml:"docker"`

	// Process is the configuration for the process launcher
	Process *ProcessLauncherConfig `json:"process" yaml:"process"`
}

// Config holds the configuration for a sigma server
type Config struct {
	// Server is the configurtaion for the sigma server
	Server SigmaServerConfig `json:"server" yaml:"server"`

	// Nodes is the configuration for the node server
	Nodes NodeServerConfig `json:"nodeServer" yaml:"nodeServer"`

	// Launchers holds launcher configuration values
	Launchers Launcher `json:"launcher" yaml:"launcher"`
}

// Valid checks if the configuration is valid
func (c Config) Valid() error {
	if c.Launchers.Docker == nil && c.Launchers.Process == nil {
		return errors.New("at least one launcher needs to be configured")
	}

	types := 0

	if c.Launchers.Docker != nil {
		for range c.Launchers.Docker.Types {
			types++
		}
	}

	if c.Launchers.Process != nil {
		for range c.Launchers.Process.Types {
			types++
		}
	}

	if types == 0 {
		return errors.New("no execution types configured")
	}

	return nil
}

// WriteJSON writes the configuration JSON encoded to w
func (c Config) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)

	return encoder.Encode(c)
}

// WriteYAML writes the configuration YAML encoded to w
func (c Config) WriteYAML(w io.Writer) error {
	blob, err := yaml.Marshal(c)
	if err != nil {
		return nil
	}

	n, err := w.Write(blob)

	if n != len(blob) && err != nil {
		return err
	}

	return nil
}

// ReadJSON read the configuartion from a JSON file
func ReadJSON(r io.Reader) (*Config, error) {
	var c Config

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

// ReadYAML reads the configuration from a YAML file
func ReadYAML(r io.Reader) (*Config, error) {
	var c Config

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
