//go:build !unix && !windows

package main

import "os"

func handoffDir() (string, bool) { return "", false }

var exitSignals = []os.Signal{os.Interrupt}

func exitOnSignal(os.Signal) { os.Exit(1) }
