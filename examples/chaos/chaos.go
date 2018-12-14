package main

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/goresilience/chaos"
)

func errorOnOddMinute(ctx context.Context) error {
	minute := time.Now().Minute()
	if minute%2 != 0 {
		return fmt.Errorf("error because %d minute is odd", minute)
	}

	return nil
}

func main() {
	var chaosctrl chaos.Injector
	chaosctrl.SetLatency(200 * time.Millisecond)
	chaosctrl.SetErrorPercent(15)

	cmd := chaos.New(chaos.Config{
		Injector: &chaosctrl,
	})

	errs := make(chan error)
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			go func() {
				errs <- cmd.Run(context.TODO(), func(ctx context.Context) error {
					fmt.Printf("[+] good\n")
					return nil
				})
			}()
		}
	}()

	for err := range errs {
		if err != nil {
			fmt.Printf("[!] err: %s\n", err)
		}
	}
}
