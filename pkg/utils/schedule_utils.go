package utils

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// NextRunTime calculates the next scheduled run time for a given cron schedule string.
// The schedule string should be in the standard cron format (6 fields: second, minute, hour, day-of-month, month, day-of-week).
func NextRunTime(cronSchedule string) (time.Time, error) {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronSchedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron schedule '%s': %w", cronSchedule, err)
	}

	return schedule.Next(time.Now()), nil
}
