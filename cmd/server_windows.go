//go:build windows
// +build windows

package main

import "os"

func addSignals(sigs []os.Signal) []os.Signal {
	return sigs
}
