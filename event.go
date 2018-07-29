package buttonoff

import (
	"time"
)

type Event struct {
	HWAddr    string
	Timestamp time.Time
}
