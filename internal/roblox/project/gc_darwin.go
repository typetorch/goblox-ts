package project

import "golang.org/x/sys/unix"

// physicalMemory is sysctl hw.memsize, or 0 if it can't be read.
func physicalMemory() uint64 {
	n, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return n
}
