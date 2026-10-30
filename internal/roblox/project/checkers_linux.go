package project

import (
	"os"
	"path/filepath"
)

// physicalCores counts the distinct cores of the online CPUs from sysfs, or returns 0 if it can't.
func physicalCores() int {
	for _, name := range []string{"core_cpus_list", "thread_siblings_list"} {
		files, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/topology/" + name)
		cores := map[string]bool{}
		for _, f := range files {
			if siblings, err := os.ReadFile(f); err == nil {
				cores[string(siblings)] = true
			}
		}
		if len(cores) > 0 {
			return len(cores)
		}
	}
	return 0
}
