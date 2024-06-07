package duration

import (
	"fmt"
	"strconv"
	"time"
)

func ParseDuration(duration string) (time.Duration, error) {
	unitMap := map[string]time.Duration{
		"s": time.Second,
		"m": time.Minute,
		"h": time.Hour,
		"d": 24 * time.Hour,
		"w": 7 * 24 * time.Hour,
		"M": 30 * 24 * time.Hour,
		"y": 365 * 24 * time.Hour,
	}
	var value string
	var unit string
	for i := len(duration) - 1; i >= 0; i-- {
		if _, ok := unitMap[duration[i:]]; ok {
			value = duration[:i]
			unit = duration[i:]
			break
		}
	}
	if unit == "" {
		return 0, fmt.Errorf("invalid duration format: %s", duration)
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", value)
	}
	return time.Duration(v * float64(unitMap[unit])), nil
}

func RelativeAge(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	} else if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dw", int(duration.Hours()/(7*24)))
	} else if duration < 365*24*time.Hour {
		return fmt.Sprintf("%dM", int(duration.Hours()/(30*24)))
	} else {
		return fmt.Sprintf("%dy", int(duration.Hours()/(365*24)))
	}
}
