package helper

import (
	"context"
	"log"
	"math"
	"time"
)

type RetryConfig struct {
	Count int
	Delay time.Duration
}

// Effector will be the function that is called by the Retry function
type Effector func(context.Context) error

// Retry will retry the run the effector function in an exponential backoff
func Retry(effector Effector, rc RetryConfig) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		for r := 1; ; r++ {
			err := effector(ctx)
			if err == nil || r > rc.Count {
				return err
			}

			delay := getExpBackoff(rc.Delay, r)
			log.Printf("Effector call failed, retrying in %v", delay)

			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
}

// calculate the exponential delay for a given iteration
// first iteration is 1
func getExpBackoff(initialDelay time.Duration, iteration int) time.Duration {
	if iteration <= 1 {
		return initialDelay
	}
	return time.Duration(math.Pow(2, float64(iteration-1))) * initialDelay
}
