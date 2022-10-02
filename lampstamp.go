package lampstamp

import "sync"

// MapStamp stores timestamp in a sized map. The maximum size of the map, m, should be determined
// upon creation.
type MapStamp struct {
	// m is a map that stores timestamps by given key
	m map[string]int64
	// mutex to lock when read/write to m
	l sync.RWMutex
	// holds the push history of m so that it can delete old keys from m.
	b *StampBuffer
}

const defaultSize = 4096

// NewMapStamp creates default MapStamp
func NewMapStamp() *MapStamp {
	return NewMapStampSize(defaultSize)
}

// NewMapStampSize creates MapStamp with size.
func NewMapStampSize(size int64) *MapStamp {
	return &MapStamp{
		m: make(map[string]int64),
		b: NewStampBuffer(size),
	}
}

const defaultTimestamp = 0

// Get retreives a timestamp of key.
func (ts *MapStamp) Get(key string) int64 {
	ts.l.RLock()
	defer ts.l.RUnlock()

	if val, ok := ts.m[key]; ok {
		return val
	}

	return defaultTimestamp
}

// Inc increments a timestamp of key if exists. Otherwise it increments defaultTimestamp
// then stores it. It returns the incremented value.
// It deletes oldest key from underlying m if the size of m is reached to the capacity of underlying b.
func (ts *MapStamp) Inc(key string) int64 {
	ts.l.Lock()
	defer ts.l.Unlock()

	var val int64
	var ok bool
	if val, ok = ts.m[key]; !ok {
		val = defaultTimestamp
	}

	val++
	ts.m[key] = val

	if !ok {
		if popped, err := ts.b.PopIfFullThenPush(key); err == nil {
			delete(ts.m, popped)
		}
	}

	return val
}

// Tick compares a timestamp of key with requestTimestamp to find maximum timestamp.
// Then it increments the maximum value by 1.
// Then it stores the new timestamp to underlying m.
// Then it deletes oldest key from underlying m if the size of m is full(capacity of underlying b).
// Then it returns the new timestamp.
func (ts *MapStamp) Tick(key string, requestTimestamp int64) int64 {
	ts.l.Lock()
	defer ts.l.Unlock()

	var val int64
	var ok bool
	if val, ok = ts.m[key]; !ok {
		val = defaultTimestamp
	}

	val = max(val, requestTimestamp)
	val++
	ts.m[key] = val

	if !ok {
		if popped, err := ts.b.PopIfFullThenPush(key); err == nil {
			delete(ts.m, popped)
		}
	}

	return val
}

func max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}
