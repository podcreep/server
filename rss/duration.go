package rss

import (
	"strconv"
	"strings"
)

// parseDuration takes a duration string in the format of "HH:MM:SS" or "MM:SS" or "SS" and returns
// the duration in seconds as an int32. If the duration string is invalid, it returns 0.
func parseDuration(duration string) int32 {
	parts := strings.Split(duration, ":")
	if len(parts) == 1 {
		d, _ := strconv.Atoi(parts[0])
		return int32(d)
	}
	if len(parts) == 2 {
		min, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.Atoi(parts[1])
		return int32(min*60 + sec)
	}
	if len(parts) == 3 {
		hour, _ := strconv.Atoi(parts[0])
		min, _ := strconv.Atoi(parts[1])
		sec, _ := strconv.Atoi(parts[2])
		return int32(hour*3600 + min*60 + sec)
	}
	return 0
}
