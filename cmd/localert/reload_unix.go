//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func watchReloadSignals(ctx context.Context, restartRequests chan<- struct{}) {
	reloadSignals := make(chan os.Signal, 2)
	signal.Notify(reloadSignals, syscall.SIGUSR1)
	go func() {
		for {
			select {
			case <-reloadSignals:
				select {
				case restartRequests <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
