//go:build !linux && !darwin

package project

// physicalMemory returns 0 (unknown): TuneGC then sets no memory limit.
func physicalMemory() uint64 { return 0 }
