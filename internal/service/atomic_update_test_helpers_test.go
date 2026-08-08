package service

import (
	"sync"
	"testing"
)

func runConcurrent(t *testing.T, n int, fn func(i int) error) []error {
	t.Helper()

	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
	return errs
}

func assertNoError(t *testing.T, errs []error) {
	t.Helper()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
}
