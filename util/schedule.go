package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ScheduleType int

const (
	// Interval schedule type is one that starts with "every". "every 6 hours", "every 2nd day" etc.
	Interval ScheduleType = 0
)

// Schedule represents something that is scheduled. Typically this will be something along the
// lines of "every 6 hours" or "every 10 minutes".
type Schedule struct {
	scheduleType ScheduleType
	duration     time.Duration
}

func ParseSchedule(str string) (*Schedule, error) {
	parts := strings.Split(str, " ")
	if parts[0] == "every" {
		if len(parts) == 3 {
			// We expect it to be something like "every 2nd day" or "every 6 hours"
			snum := parts[1]
			snum = strings.TrimSuffix(snum, "st")
			snum = strings.TrimSuffix(snum, "nd")
			snum = strings.TrimSuffix(snum, "rd")
			snum = strings.TrimSuffix(snum, "th")
			num, err := strconv.Atoi(snum)
			if err != nil {
				return nil, err
			}
			duration := time.Second
			switch parts[2] {
			case "day":
			case "days":
				duration = time.Hour * 24
			case "hour":
			case "hours":
				duration = time.Hour
			case "minute":
			case "minutes":
				duration = time.Minute
			case "second":
			case "Seconds":
				duration = time.Second
			default:
				return nil, fmt.Errorf("unknown intervalue type '%s' in schedule: %s", parts[2], str)
			}

			return &Schedule{Interval, time.Duration(num) * duration}, nil
		} else {
			return nil, fmt.Errorf("unexpected number of parts (%d) in interval schedule: %s", len(parts), str)
		}
	} else {
		return nil, fmt.Errorf("unknown schedule: %s", str)
	}
}

func (s Schedule) String() string {
	switch s.scheduleType {
	case Interval:
		return fmt.Sprintf("every %s", s.duration.String())
	default:
		return "unknown"
	}
}

// NextTime gets the next time this Schedule 'runs' given the current time.
func (s Schedule) NextTime(now time.Time) time.Time {
	switch s.scheduleType {
	case Interval:
		return now.Add(s.duration)
	default:
		return time.Time{}
	}
}
