package smtp

import (
	"time"
)

func Processing(start, end time.Time) int64 {
	return end.Sub(start).Nanoseconds() / int64(time.Millisecond)
}
