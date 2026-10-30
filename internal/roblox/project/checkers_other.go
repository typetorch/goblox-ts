//go:build !linux && !darwin

package project

// physicalCores returns 0 (unknown): DefaultCheckers then assumes half the threads are cores.
func physicalCores() int { return 0 }
