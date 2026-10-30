package project

import "runtime"

// DefaultCheckers is the type checker count used when LoadOptions.Checkers is 0.
//
// typescript-go defaults to 4. Every checker builds its own type caches, so a checker without a
// physical core to itself mostly duplicates work and memory: on a 2-core/4-thread host, 2 checkers
// built gnome-game 5% faster than 4 with 28% less CPU and 17% less memory (BENCHMARKS-espresso.md).
// So it is min(4, physical cores), with 4 kept on hosts with 8 or more threads. When the physical
// core count isn't known (only Linux and macOS report it here), half the threads are assumed to be
// cores. GOMAXPROCS counts as the thread count, so a CPU limit (or GOMAXPROCS=2) lowers it too.
func DefaultCheckers() int {
	threads := runtime.GOMAXPROCS(0)
	if threads >= 8 {
		return 4
	}
	cores := physicalCores()
	if cores <= 0 {
		cores = threads / 2
	}
	return max(1, min(4, cores, threads))
}
