package main

import "sync"

type Timestamp struct {
	m map[string]int64
	l sync.RWMutex
}

func NewTimestamp() *Timestamp {
	return &Timestamp{
		m: make(map[string]int64),
	}
}

const defaultTimestamp = 0

func (ts *Timestamp) Get(key string) int64 {
	ts.l.RLock()
	defer ts.l.RUnlock()

	if val, ok := ts.m[key]; ok {
		return val
	}

	return defaultTimestamp
}

func (ts *Timestamp) Inc(key string) int64 {
	ts.l.Lock()
	defer ts.l.Unlock()

	var val int64
	var ok bool
	if val, ok = ts.m[key]; !ok {
		val = defaultTimestamp
	}

	val++
	ts.m[key] = val
	return val
}

func (ts *Timestamp) Tick(key string, requestTimestamp int64) int64 {
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
	return val
}

func max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}
