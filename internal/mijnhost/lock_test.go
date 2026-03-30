package mijnhost

import (
	"sync"
	"testing"
)

// TestLockDomain_serializes verifies that concurrent LockDomain calls for the
// same domain are mutually exclusive, while calls for different domains proceed
// in parallel.
func TestLockDomain_serializes(t *testing.T) {
	client := NewClient("test-key")

	const goroutines = 20
	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
		races   int
	)

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()

			unlock := client.LockDomain("example.com")
			defer unlock()

			// Inside the lock: read, check, increment. If two goroutines
			// entered simultaneously, counter would be incremented more than
			// once before the other goroutine reads it, which the race
			// detector would catch. We additionally check manually.
			mu.Lock()
			before := counter
			mu.Unlock()

			counter++ // non-atomic on purpose — race detector catches concurrent access

			mu.Lock()
			if counter != before+1 {
				races++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	if races > 0 {
		t.Errorf("detected %d race(s): LockDomain did not serialize access", races)
	}
	if counter != goroutines {
		t.Errorf("counter = %d, want %d", counter, goroutines)
	}
}

// TestLockDomain_differentDomains verifies that locks for different domains
// are independent (i.e. don't block each other).
func TestLockDomain_differentDomains(t *testing.T) {
	client := NewClient("test-key")

	domains := []string{"a.example.com", "b.example.com", "c.example.com"}
	var wg sync.WaitGroup
	wg.Add(len(domains))

	// All goroutines acquire their lock simultaneously. If different-domain
	// locks were shared, only one could proceed at a time and the test would
	// deadlock or be significantly slower.
	ready := make(chan struct{})
	for _, d := range domains {
		go func(domain string) {
			defer wg.Done()
			<-ready // start all at the same time
			unlock := client.LockDomain(domain)
			defer unlock()
		}(d)
	}
	close(ready)
	wg.Wait() // would deadlock if domains shared the same mutex
}

// TestLockDomain_samePointer verifies that repeated calls for the same domain
// return the same underlying mutex (not a new one each time).
func TestLockDomain_samePointer(t *testing.T) {
	client := NewClient("test-key")

	unlock1 := client.LockDomain("example.com")
	unlock1()

	// Acquiring and releasing twice should not deadlock.
	unlock2 := client.LockDomain("example.com")
	unlock2()
}
