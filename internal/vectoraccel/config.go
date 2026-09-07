package vectoraccel

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ModeOff      = "off"
	ModeAuto     = "auto"
	ModeRequired = "required"
)

// Config controls the optional VectorScan learning accelerator. Coraza remains
// authoritative in all modes; required only means the native scanner must be
// available when an accelerated site is built.
type Config struct {
	Mode                   string  `json:"mode,omitempty"`
	StatePath              string  `json:"state_path,omitempty"`
	MinSamples             int64   `json:"min_samples,omitempty"`
	MinCorazaMatches       int64   `json:"min_coraza_matches,omitempty"`
	MinLearningSec         int     `json:"min_learning_sec,omitempty"`
	VerificationSampleRate float64 `json:"verification_sample_rate,omitempty"`
}

func Defaults() Config {
	return Config{Mode: ModeOff, StatePath: "/var/lib/waf-proxy/vector-learning.json", MinSamples: 100000, MinCorazaMatches: 10, MinLearningSec: 3600, VerificationSampleRate: 0.01}
}

func Effective(c Config) Config {
	if c.Mode == "" {
		c.Mode = ModeOff
	}
	if c.StatePath == "" {
		c.StatePath = "/var/lib/waf-proxy/vector-learning.json"
	}
	if c.MinSamples == 0 {
		c.MinSamples = 100000
	}
	if c.MinCorazaMatches == 0 {
		c.MinCorazaMatches = 10
	}
	if c.MinLearningSec == 0 {
		c.MinLearningSec = 3600
	}
	if c.VerificationSampleRate == 0 {
		c.VerificationSampleRate = 0.01
	}
	return c
}

func Validate(c Config) error {
	c = Effective(c)
	switch c.Mode {
	case ModeOff, ModeAuto, ModeRequired:
	default:
		return fmt.Errorf("vector_acceleration.mode must be off, auto, or required (got %q)", c.Mode)
	}
	if c.MinSamples < 1 || c.MinSamples > 1_000_000_000 {
		return errors.New("vector_acceleration.min_samples must be 1..1000000000")
	}
	if c.MinCorazaMatches < 0 || c.MinCorazaMatches > c.MinSamples {
		return errors.New("vector_acceleration.min_coraza_matches must be 0..min_samples")
	}
	if c.MinLearningSec < 0 || c.MinLearningSec > int((365*24*time.Hour)/time.Second) {
		return errors.New("vector_acceleration.min_learning_sec out of range")
	}
	if c.VerificationSampleRate <= 0 || c.VerificationSampleRate > 1 {
		return errors.New("vector_acceleration.verification_sample_rate must be >0 and <=1")
	}
	if strings.ContainsAny(c.StatePath, "\x00\r\n") {
		return errors.New("vector_acceleration.state_path contains invalid characters")
	}
	return nil
}
