package main

import (
	"errors"
	"time"
)

func validateFlags(readWindow time.Duration, variantThreshold float64) error {
	if readWindow < 0 {
		return errors.New("durations must be non-negative")
	}
	if readWindow > 0 && readWindow < time.Millisecond {
		return errors.New("--read-window must be positive")
	}
	if variantThreshold <= 0 || variantThreshold >= 1 {
		return errors.New("--variant-threshold must satisfy 0 < T < 1")
	}
	return nil
}
