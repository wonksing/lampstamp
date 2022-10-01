package lampstamp

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

func TestBuffer(t *testing.T) {
	sb := NewStampBuffer(3)

	for i := 0; i < 10; i++ {
		// sb.Push(strconv.Itoa(i))
		pushed := strconv.Itoa(i)
		popped, err := sb.PopIfFullThenPush(pushed)
		if errors.Is(err, ErrNothingToPop) {
			fmt.Printf("pushed: %s, %s\n", pushed, sb)
		} else {
			fmt.Printf("popped: %s, pushed: %s, %s\n", popped, pushed, sb)
		}
	}
}
