package stats

import (
	"sync"
	"time"

	"github.com/gammazero/deque"
)

type Member struct {
	bytes     int64
	timestamp time.Time
}

// Using a mutex for thread safety since multiple functions are accessing the same data
// using sliding window + prefix sum to calculate the rate.
type RateCalculator struct {
	dq          deque.Deque[Member]
	windowBytes int64
	window      time.Duration
	mu          sync.Mutex
}

func (rc *RateCalculator) Add(bytes int64) {

	// Bucketing for round offing the calc.
	now := time.Now().Truncate(time.Second)

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.dq.Len() > 0 && rc.dq.Back().timestamp.Equal(now) {
		last := rc.dq.PopBack()
		last.bytes += bytes
		rc.dq.PushBack(last)
	} else {
		rc.dq.PushBack(Member{bytes: bytes, timestamp: now})
	}

	rc.windowBytes += bytes
	rc.Prune(now)
	
}

func (rc *RateCalculator) Prune(now time.Time) {

	for rc.dq.Len() > 0 && now.Sub(rc.dq.Front().timestamp) > rc.window {
		rc.windowBytes -= rc.dq.Front().bytes
		rc.dq.PopFront()
	}

}
