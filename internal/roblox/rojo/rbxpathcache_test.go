package rojo

import (
	"slices"
	"sync"
	"testing"
)

func TestGetRbxPathFromFilePathMemo(t *testing.T) {
	t.Parallel()
	r := newRojoResolver(nil)
	r.partitions = []PartitionInfo{{RbxPath: RbxPath{"ReplicatedStorage"}, FsPath: "/game/out"}}

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if got := r.GetRbxPathFromFilePath("/game/out/a/b.luau"); !slices.Equal(got, RbxPath{"ReplicatedStorage", "a", "b"}) {
				t.Errorf("got %v", got)
			}
		})
	}
	wg.Wait()

	// nil (no partition) stays nil on a cache hit
	for range 2 {
		if got := r.GetRbxPathFromFilePath("/elsewhere/a.luau"); got != nil {
			t.Errorf("outside every partition: got %v, want nil", got)
		}
	}

	// an empty path (the root) stays non-nil on a cache hit
	root := newRojoResolver(nil)
	root.partitions = []PartitionInfo{{RbxPath: RbxPath{}, FsPath: "/game"}}
	for range 2 {
		if got := root.GetRbxPathFromFilePath("/game"); got == nil || len(got) != 0 {
			t.Errorf("partition root: got %#v, want empty and non-nil", got)
		}
	}

	// an append by a caller doesn't reach the cached slice
	first := r.GetRbxPathFromFilePath("/game/out/a/b.luau")
	_ = append(first, "x")
	if got := r.GetRbxPathFromFilePath("/game/out/a/b.luau"); cap(got) != len(got) || !slices.Equal(got, RbxPath{"ReplicatedStorage", "a", "b"}) {
		t.Errorf("cached slice changed: %v (cap %d)", got, cap(got))
	}
}
