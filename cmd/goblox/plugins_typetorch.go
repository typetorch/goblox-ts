//go:build typetorch

package main

// Links the Go port of @typetorch/transformer (github.com/typetorch/transformer/go), which registers
// itself in pkg/goblox's plugin registry. Build with a go.work that uses both modules:
//
//	go work init ./goblox-ts ./typetorch-transformer/go
//	go build -tags typetorch -o goblox.exe ./goblox-ts/cmd/goblox
import _ "github.com/typetorch/transformer/go"
