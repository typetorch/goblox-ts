//go:build typetorch

package main

// Without plugins_typetorch.go (see plugins_typetorch.go.in), -tags typetorch would silently build a
// goblox that has no transformer. This makes it a compile error instead.
const _ = typetorchLinked
