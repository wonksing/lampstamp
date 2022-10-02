package lampstamp

import "sync"

type Lampstamp struct {
	m map[string]int64
	l sync.RWMutex
	b *StampBuffer
}

func NewLampstamp() *Lampstamp {
	return NewLampstampSize(1024)
}

func NewLampstampSize(size int64) *Lampstamp {
	return &Lampstamp{
		m: make(map[string]int64),
		b: NewStampBuffer(size),
	}
}

const defaultTimestamp = 0

func (ts *Lampstamp) Get(key string) int64 {
	ts.l.RLock()
	defer ts.l.RUnlock()

	if val, ok := ts.m[key]; ok {
		return val
	}

	return defaultTimestamp
}

func (ts *Lampstamp) Inc(key string) int64 {
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

func (ts *Lampstamp) Tick(key string, requestTimestamp int64) int64 {
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
