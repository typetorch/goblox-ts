package project

import "golang.org/x/sys/unix"

// physicalMemory is the total RAM from sysinfo(2), or 0 if it can't be read.
func physicalMemory() uint64 {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return 0
	}
	return uint64(info.Totalram) * uint64(info.Unit) //nolint:unconvert // Totalram is a uint32 on 32-bit platforms
}
