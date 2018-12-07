package circuitbreaker

import (
	"sync"
	"time"
)

// recorder knows how to record the request and errors for a circuitbreaker.
type recorder interface {
	inc(err error)
	reset()
	errorRate() float64
	totalRequests() float64
}

type bucket struct {
	total float64
	errs  float64
}

// bucketsWindow records the data in N buckets of T duration, the N buckets
// will be the window of recording.
type bucketWindow struct {
	// Used to keep track of the oldest bucket replace the oldest bucket
	// on the window, this can be made because we don't ned order to get
	// totals and percerts of all the window.
	nextIndexToReplace int
	windowSize         int
	window             []*bucket
	currentBucket      *bucket
	mu                 sync.Mutex
}

func newBucketWindow(bucketQuantity int, bucketDuration time.Duration) recorder {
	// If no bucketNumber then act as a uniquecounter.
	if bucketQuantity == 0 {
		bucketQuantity = 1
	}

	b := &bucketWindow{
		windowSize: bucketQuantity,
	}
	b.reset()

	// Only move the window if we have a duration for the buckets.
	if bucketDuration != 0 {
		go b.windowSlider(bucketDuration)
	}

	return b
}

// windowSlider will slide the bucket moving window with the duration and
// the current time by replacing the oldest bucket with a new one and setting
// the latest bucket to this one.
func (b *bucketWindow) windowSlider(bucketDuration time.Duration) {
	ticker := time.NewTicker(bucketDuration)
	for range ticker.C {
		b.mu.Lock()

		// Create a new bucket and replace the oldest one.
		bucket := &bucket{}
		b.window[b.nextIndexToReplace] = bucket
		b.currentBucket = bucket

		// Leave ready the next one to be replaced
		b.nextIndexToReplace++
		// If we have passed the lenght start form 0 again.
		if b.nextIndexToReplace >= len(b.window) {
			b.nextIndexToReplace = 0
		}

		b.mu.Unlock()
	}
}

func (b *bucketWindow) inc(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.currentBucket.total++
	if err != nil {
		b.currentBucket.errs++
	}
}

func (b *bucketWindow) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Init the bucket window.
	window := make([]*bucket, b.windowSize)
	for i := 0; i < b.windowSize; i++ {
		window[i] = &bucket{}
	}
	b.window = window
	// Set the current bucket.
	b.currentBucket = window[0]
	// Set the next bucket position.
	b.nextIndexToReplace = 1
}

func (b *bucketWindow) errorRate() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	var total float64
	var errs float64

	for _, bucket := range b.window {
		total += bucket.total
		errs += bucket.errs
	}
	return errs / total
}

func (b *bucketWindow) totalRequests() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	var total float64
	for _, bucket := range b.window {
		total += bucket.total
	}
	return total
}
