package project

import (
	"os"
	"runtime/debug"
)

// gcPercent is goblox's GOGC. A build allocates a lot and keeps most of it to the end, so collecting less
// often pays: a fresh gnome-game build (647 files, 2-core laptop) took 24% less CPU and 18% less wall time
// at 400 than at Go's 100, for 30% more peak memory (1.19 GB against 0.91 GB).
const gcPercent = 400

// TuneGC sets gcPercent and a soft memory limit of a third of the physical memory (at least 1 GiB), so the
// GC collects sooner on a machine that has little. GOGC or GOMEMLIMIT in the environment leave Go's own
// settings alone.
func TuneGC() {
	if gcFromEnvironment() {
		return
	}
	debug.SetGCPercent(gcPercent)
	if total := physicalMemory(); total > 0 {
		debug.SetMemoryLimit(int64(max(total/3, 1<<30)))
	}
}

// idleGCPercent is the GOGC of a watcher between builds: Go's default. The polls make a little garbage
// all the time, which at gcPercent would pile up for a long time on top of the session's state.
const idleGCPercent = 100

// SetBuildGC switches between gcPercent, while a watch build runs, and idleGCPercent between builds. GOGC
// or GOMEMLIMIT in the environment leave Go's own settings alone, as in TuneGC.
func SetBuildGC(building bool) {
	if gcFromEnvironment() {
		return
	}
	if building {
		debug.SetGCPercent(gcPercent)
	} else {
		debug.SetGCPercent(idleGCPercent)
	}
}

func gcFromEnvironment() bool {
	return os.Getenv("GOGC") != "" || os.Getenv("GOMEMLIMIT") != ""
}
