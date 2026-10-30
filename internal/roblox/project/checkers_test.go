package project

import (
	"runtime"
	"testing"
)

func TestDefaultCheckers(t *testing.T) { //nolint:paralleltest // sets GOMAXPROCS.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(0))
	for _, threads := range []int{1, 2, 3, 4, 6, 8, 12, 64} {
		runtime.GOMAXPROCS(threads)
		got := DefaultCheckers()
		if got < 1 || got > 4 || got > threads {
			t.Errorf("GOMAXPROCS=%d: DefaultCheckers() = %d, want 1..min(4, threads)", threads, got)
		}
		if threads >= 8 && got != 4 {
			t.Errorf("GOMAXPROCS=%d: DefaultCheckers() = %d, want 4", threads, got)
		}
	}
	t.Logf("physical cores: %d", physicalCores())
}
