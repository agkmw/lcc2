package sysinfo

import "time"

// FormatBytes renders a byte count in human readable units.
func FormatBytes(b float64) string {
	const unit = 1024
	if b < unit {
		return trimZero(b) + " B"
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	div, exp := float64(unit), 0
	for n := b / unit; n >= unit && exp < len(units)-1; n /= unit {
		div *= unit
		exp++
	}
	return trimZero(b/div) + " " + units[exp]
}

// FormatUptime renders a duration like "3d 4h" or "12m".
func FormatUptime(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return itoa(int(d.Hours()/24)) + "d " + itoa(int(d.Hours())%24) + "h"
	case d >= time.Hour:
		return itoa(int(d.Hours())) + "h " + itoa(int(d.Minutes())%60) + "m"
	case d >= time.Minute:
		return itoa(int(d.Minutes())) + "m"
	default:
		return itoa(int(d.Seconds())) + "s"
	}
}

// FormatRate renders bytes/sec with a "/s" suffix.
func FormatRate(bps float64) string { return FormatBytes(bps) + "/s" }

func trimZero(f float64) string {
	i := int(f * 10)
	whole, frac := i/10, i%10
	if frac == 0 {
		return itoa(whole)
	}
	return itoa(whole) + "." + itoa(frac)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
