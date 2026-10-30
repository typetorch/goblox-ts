package project

import "golang.org/x/sys/unix"

// physicalCores is sysctl hw.physicalcpu, or 0 if it can't be read.
func physicalCores() int {
	n, err := unix.SysctlUint32("hw.physicalcpu")
	if err != nil {
		return 0
	}
	return int(n)
}
