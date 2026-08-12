package main

import (
	"context"
	"suda-forge/internal/events"
	"suda-forge/internal/productexperience"
)

func forwardProductActivity(ctx context.Context, bus *events.Bus, log *productexperience.ActivityLog) {
	if bus == nil || log == nil {
		return
	}
	ch := bus.Subscribe(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			log.AppendEvent(e)
		}
	}
}
