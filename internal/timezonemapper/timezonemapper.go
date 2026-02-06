package timezonemapper

// LatLngToTimezoneString maps coordinates to an IANA timezone name.
// It uses lightweight geographic rules suitable for offline usage.
func LatLngToTimezoneString(latitude float64, longitude float64) string {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return "unknown"
	}

	// Continental US (rough split by longitude).
	if latitude >= 24 && latitude <= 50 && longitude >= -125 && longitude <= -66 {
		switch {
		case longitude >= -82.5:
			return "America/New_York"
		case longitude >= -97.5:
			return "America/Chicago"
		case longitude >= -112.5:
			return "America/Denver"
		default:
			return "America/Los_Angeles"
		}
	}

	// Alaska and Hawaii.
	if latitude >= 51 && latitude <= 72 && longitude >= -170 && longitude <= -129 {
		return "America/Anchorage"
	}
	if latitude >= 18 && latitude <= 23 && longitude >= -161 && longitude <= -154 {
		return "Pacific/Honolulu"
	}

	// Basic Europe coverage.
	if latitude >= 35 && latitude <= 72 && longitude >= -10 && longitude <= 2 {
		return "Europe/London"
	}
	if latitude >= 35 && latitude <= 72 && longitude > 2 && longitude <= 30 {
		return "Europe/Berlin"
	}
	if latitude >= 35 && latitude <= 72 && longitude > 30 && longitude <= 45 {
		return "Europe/Moscow"
	}

	// Fallback: return unknown to let caller skip offset update.
	return "unknown"
}
