package hosting

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ValidateAgentCron(expr string, minIntervalSec int) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cron must have 5 fields")
	}
	if expr == "* * * * *" {
		return fmt.Errorf("cron interval is too frequent")
	}
	minute := fields[0]
	if strings.HasPrefix(minute, "*/") {
		n, err := strconv.Atoi(strings.TrimPrefix(minute, "*/"))
		if err != nil || n*60 < minIntervalSec {
			return fmt.Errorf("cron interval is below the minimum of %d seconds", minIntervalSec)
		}
	}
	return nil
}

func NextCronTime(expr string, from time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("cron must have 5 fields")
	}
	t := from.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 60*24*370; i++ {
		if cronMatch(fields, t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no cron match")
}

func cronMatch(fields []string, t time.Time) bool {
	return cronFieldMatch(fields[0], t.Minute()) &&
		cronFieldMatch(fields[1], t.Hour()) &&
		cronFieldMatch(fields[2], t.Day()) &&
		cronFieldMatch(fields[3], int(t.Month())) &&
		cronFieldMatch(fields[4], int(t.Weekday()))
}

func cronFieldMatch(field string, value int) bool {
	if field == "*" {
		return true
	}
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil || n <= 0 {
			return false
		}
		return value%n == 0
	}
	for _, part := range strings.Split(field, ",") {
		n, err := strconv.Atoi(part)
		if err == nil && n == value {
			return true
		}
	}
	return false
}
