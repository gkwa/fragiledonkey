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

	value := duration[:len(duration)-1]
	unit := duration[len(duration)-1:]

	if _, ok := unitMap[unit]; !ok {
		return 0, fmt.Errorf("invalid duration unit: %s", unit)
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", value)
	}

	return time.Duration(intValue) * unitMap[unit], nil
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
