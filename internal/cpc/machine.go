// Package cpc coordinates the emulated CPC machine.
package cpc

import "cpcgo/internal/rom"

// Model identifies the CPC model being emulated.
type Model string

const Model6128 Model = "6128"

// Config contains startup configuration for a CPC machine.
type Config struct {
	Model Model
	ROMs  rom.Image
	Disk  string
	Scale int
}

// Machine is the top-level emulator state.
type Machine struct {
	config Config
}

// New constructs a machine from validated configuration.
func New(config Config) *Machine {
	return &Machine{config: config}
}

// Config returns the machine startup configuration.
func (m *Machine) Config() Config {
	return m.config
}
