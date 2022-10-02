package lampstamp

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLampstamp(t *testing.T) {
	lt := NewMapStampSize(5)
	for i := 0; i < 10; i++ {
		key := strconv.Itoa(i)
		lt.Tick(key, 1)
	}

	lt.Tick("0", 1)
	lt.Tick("0", 1)
	lt.Tick("0", 1)
	lt.Tick("10", 1)
	lt.Tick("11", 1)
	lt.Tick("10", 1)
	lt.Tick("12", 1)

	expectedMap := map[string]int64{
		"0": 4, "10": 3, "11": 2, "12": 2,
	}
	expectedBuffer := []string{"0", "10", "11", "12", "9"}
	assert.EqualValues(t, expectedMap, lt.m)
	assert.EqualValues(t, expectedBuffer, lt.b.b)
}

func BenchmarkLampstamp(b *testing.B) {
	lt := NewMapStampSize(102400)
	benchmarks := []string{"1", "2"}
	for _, bm := range benchmarks {
		b.Run(bm, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := uuid.New().String()
				lt.Tick(id, 2)
			}
		})
	}
}

func BenchmarkLampstampParallel(b *testing.B) {
	lt := NewMapStampSize(102400)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := uuid.New().String()
			lt.Tick(id, 2)
		}
	})

}
