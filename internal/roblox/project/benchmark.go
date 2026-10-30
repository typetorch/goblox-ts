package project

// Port of Shared/util/benchmark.ts: with --verbose, each build step prints its name followed by
// ` ( <n> ms )`, like roblox-ts. goblox times more steps than roblox-ts (program creation, the type
// check, the transform and the render), because it runs them as separate parallel phases instead of
// one sequential per-file loop.

import (
	"fmt"
	"time"
)

func benchmarkStart(name string) time.Time {
	LogService.Write(name)
	return time.Now()
}

func benchmarkEnd(startTime time.Time) {
	LogService.Write(fmt.Sprintf(" ( %d ms )\n", time.Since(startTime).Milliseconds()))
}

func benchmarkSync(name string, callback func()) {
	startTime := benchmarkStart(name)
	callback()
	benchmarkEnd(startTime)
}

// benchmarkIfVerbose is benchmark.ts benchmarkIfVerbose.
func benchmarkIfVerbose(name string, callback func()) {
	if LogService.Verbose {
		benchmarkSync(name, callback)
	} else {
		callback()
	}
}
