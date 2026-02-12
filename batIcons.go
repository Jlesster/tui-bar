package main

func getBatteryIcon(level int, state string) string {
	if state == "charging" {
		return "󰂄"
	}

	switch {
	case level >= 90:
		return "󰁹" // Full
	case level >= 80:
		return "󰂂"
	case level >= 70:
		return "󰂁"
	case level >= 60:
		return "󰂀"
	case level >= 50:
		return "󰁿"
	case level >= 40:
		return "󰁾"
	case level >= 30:
		return "󰁽"
	case level >= 20:
		return "󰁼"
	case level >= 10:
		return "󰁻"
	default:
		return "󰁺" // Critical
	}
}

func getNetworkIcon(state string) string {
	if state == "connected" {
		return "󰖩 "
	}
	return "󰖪 "
}
