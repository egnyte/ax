package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx := context.Background()

	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			fmt.Println("Ctrl-c")
			cancel()
		case <-ctx.Done():
			fmt.Println("Done")
		}
	}()

	fmt.Println("Sleepin")

	time.Sleep(time.Second * 5)
}
