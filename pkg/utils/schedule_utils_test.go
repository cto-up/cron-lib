package utils

import (
	"testing"
	"time"
)

func TestNextRunTime(t *testing.T) {
	tests := []struct {
		name         string
		cronSchedule string
		// nextRunTime will be compared with the result, allowing for a small time delta.
		// For schedules like "every minute", we will check if the result is in the next minute.
		// For now, we will just check if the function returns an error for invalid schedules.
		expectError bool
		// delta is the maximum allowable difference between the expected next run time and the actual next run time
		// for schedules that are easily predictable (e.g. schedules that run every second or minute).
		// For more complex schedules, we will rely on checking if it falls within a certain minute/hour.
		delta time.Duration
	}{
		{
			name:         "Valid schedule - Every second",
			cronSchedule: "* * * * * *", // Every second
			expectError:  false,
			delta:        2 * time.Second, // Allow 2 seconds difference
		},
		{
			name:         "Valid schedule - Every minute",
			cronSchedule: "0 * * * * *", // Every minute
			expectError:  false,
			delta:        1 * time.Minute, // Allow 1 minute difference for minute-level precision
		},
		{
			name:         "Invalid schedule - Too few fields",
			cronSchedule: "* * * * *",
			expectError:  true,
		},
		{
			name:         "Invalid schedule - Too many fields",
			cronSchedule: "* * * * * * *",
			expectError:  true,
		},
		{
			name:         "Invalid schedule - Malformed expression",
			cronSchedule: "invalid cron string",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextTime, err := NextRunTime(tt.cronSchedule)

			if tt.expectError {
				if err == nil {
					t.Errorf("NextRunTime() expected an error for schedule '%s', but got none", tt.cronSchedule)
				}
			} else {
				if err != nil {
					t.Errorf("NextRunTime() for schedule '%s' returned an unexpected error: %v", tt.cronSchedule, err)
				}

				// For valid schedules, check if the calculated next run time is in the near future.
				// This is a basic check. More precise checks would involve mocking time or
				// carefully calculating the exact expected next time, which can be complex
				// for all possible cron expressions.
				now := time.Now()
				if nextTime.Before(now) {
					t.Errorf("NextRunTime() for schedule '%s' returned a time in the past: %s (now: %s)", tt.cronSchedule, nextTime, now)
				}
				// For schedules with second/minute precision, check if the next time is within a reasonable delta.
				// This is a simplified check for testing purposes.
				if tt.delta > 0 {
					diff := nextTime.Sub(now)
					if diff > tt.delta+1*time.Second { // Add a small buffer to delta
						t.Logf("NextRunTime() for schedule '%s' returned %s, expected within %s of now (%s). Difference: %s", tt.cronSchedule, nextTime, tt.delta, now, diff)
					}
				}
			}
		})
	}
}
