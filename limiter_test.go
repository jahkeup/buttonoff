package buttonoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAcceptPeriod(t *testing.T) {
	type testTrial struct {
		message  string
		delay    time.Duration
		expected bool
	}
	type testCase struct {
		name   string
		period time.Duration
		trials []testTrial
	}

	tcs := []testCase{
		{
			name:   "100ms-period",
			period: duration("100ms"),
			trials: []testTrial{
				{
					message:  "should accept first",
					delay:    duration("1ms"),
					expected: true,
				},
				{
					message:  "should deny second within 10ms",
					delay:    duration("10ms"),
					expected: false,
				},
				{
					message:  "should allow after period (91+10>100)",
					delay:    duration("91ms"),
					expected: true,
				},
			},
		},
		{
			name:   "100ms-period-repeats",
			period: duration("100ms"),
			trials: []testTrial{
				{
					message:  "should accept first",
					delay:    duration("1ms"),
					expected: true,
				},
				{
					message:  "should deny second within 10ms",
					delay:    duration("10ms"),
					expected: false,
				},
				{
					message:  "should allow after period (91+10>100)",
					delay:    duration("91ms"),
					expected: true,
				},
				{
					message:  "let fill, should accept",
					delay:    duration("201ms"),
					expected: true,
				},
				{
					message:  "shouldn't burst",
					delay:    duration("10ms"),
					expected: false,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewPressRateLimiter(tc.period)

			for _, trial := range tc.trials {
				// t.Logf("Delaying %s for test behavior.", trial.delay)
				time.Sleep(trial.delay)
				assert.Equal(t, trial.expected, limiter.Accept("key"), trial.message)
			}
		})
	}
}

func duration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
