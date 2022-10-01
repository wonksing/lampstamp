package lampstamp

import (
	"errors"
	"fmt"
)

type StampBuffer struct {
	b []string
	r int64
	w int64
}

func NewStampBuffer(size int64) *StampBuffer {
	return &StampBuffer{
		b: make([]string, size),
		r: 0,
		w: 0,
	}
}

func (s *StampBuffer) String() string {
	return fmt.Sprintf("read: %d, write: %d, %v, ", s.r, s.w, s.b)
}

var ErrNothingToPop = errors.New("nothing to pop")

func (s *StampBuffer) PopIfFullThenPush(val string) (pop string, err error) {
	if s.r > s.w || s.w == s.Cap()-1 {
		pop = s.b[s.r]
	} else {
		err = ErrNothingToPop
	}
	s.Push(val)

	return
}

func (s *StampBuffer) Push(val string) {
	s.b[s.w] = val

	s.incWritePosition()
}

var ErrIsEmpty = errors.New("buffer is empty")

func (s *StampBuffer) Pop() (string, error) {
	if s.IsEmpty() {
		return "", ErrIsEmpty
	}
	val := s.b[s.r]
	s.incReadPosition()
	return val, nil
}

func (s *StampBuffer) incWritePosition() {
	if s.w == s.Cap()-1 {
		s.w = 0
	} else {
		s.w++
	}

	if s.IsEmpty() {
		s.incReadPosition()
	}
}

func (s *StampBuffer) incReadPosition() {
	if s.r == s.Cap()-1 {
		s.r = 0
	} else {
		s.r++
	}
}

func (s *StampBuffer) Cap() int64 {
	return int64(cap(s.b))
}

func (s *StampBuffer) IsEmpty() bool {
	return s.r == s.w
}
