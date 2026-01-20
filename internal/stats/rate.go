package stats

import (
	"sync"
	"time"

	"github.com/gammazero/deque"
)

// Sample represents a data point for rate calculation
type Sample struct {
	Bytes     int64
	Timestamp time.Time
}

// RateCalculator calculates download speed using a sliding window
type RateCalculator struct {
	samples     deque.Deque[Sample]
	windowBytes int64
	window      time.Duration
	mu          sync.Mutex
}

// NewRateCalculator creates a new rate calculator with the given window size
func NewRateCalculator(window time.Duration) *RateCalculator {
	return &RateCalculator{
		window: window,
	}
}

// Add records bytes downloaded at the current time
func (rc *RateCalculator) Add(bytes int64) {
	now := time.Now().Truncate(time.Second)

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Merge with last sample if same second
	if rc.samples.Len() > 0 && rc.samples.Back().Timestamp.Equal(now) {
		last := rc.samples.PopBack()
		last.Bytes += bytes
		rc.samples.PushBack(last)
	} else {
		rc.samples.PushBack(Sample{Bytes: bytes, Timestamp: now})
	}

	rc.windowBytes += bytes
	rc.Prune(now)
}

// Rate returns the current download speed in bytes per second
func (rc *RateCalculator) Rate() float64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.Prune(time.Now())

	if rc.samples.Len() == 0 {
		return 0
	}

	return float64(rc.windowBytes) / rc.window.Seconds()
}

// Prune removes samples outside the window
func (rc *RateCalculator) Prune(now time.Time) {
	for rc.samples.Len() > 0 && now.Sub(rc.samples.Front().Timestamp) > rc.window {
		rc.windowBytes -= rc.samples.Front().Bytes
		rc.samples.PopFront()
	}
}
