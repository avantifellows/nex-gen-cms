package utils

import "time"

func GetCurrentYearLast2Digits() int {
	return time.Now().Year() % 100
}
