//go:build windows

package main

import "context"

func watchReloadSignals(context.Context, chan<- struct{}) {}
