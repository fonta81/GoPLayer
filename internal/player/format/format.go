package format

import (
	"fmt"
	"time"
)

func Duration(d time.Duration) string {
	d = d.Round(time.Second)
	return fmt.Sprintf("%02d:%02d", d/time.Minute, (d%time.Minute)/time.Second)
}

func ClampDuration(val, minVal, maxVal time.Duration) time.Duration {
	return max(minVal, min(maxVal, val))
}

func Truncate(str string, maxLen int) string {
	runes := []rune(str)
	if len(runes) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

func VolumePercentage(level, minVal, maxVal float64) float64 {
	if maxVal == minVal {
		return 0.0
	}
	return max(0.0, min(1.0, (level-minVal)/(maxVal-minVal)))
}

func VolumePercentageInt(level, minVal, maxVal float64) int {
	return int(VolumePercentage(level, minVal, maxVal) * 100)
}
