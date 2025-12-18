package utils

import (
	"fmt"
	"time"
)

// ResetTime resets the time component based on the granularity specified.
// Pass "minute" to reset seconds to zero.
// Pass "hour" to reset minutes and seconds to zero.
func ResetTime(t time.Time, granularity string) time.Time {
	switch granularity {
	case "minute":
		return t.Truncate(time.Minute) // Resets seconds to zero
	case "hour":
		return t.Truncate(time.Hour) // Resets minutes and seconds to zero
	default:
		fmt.Println("Invalid granularity. Please use 'minute' or 'hour'.")
		return t
	}
}
