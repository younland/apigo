package design

// Mode is the appearance the app draws in.
type Mode int

const (
	// System follows the desktop's appearance until a fixed one is picked.
	System Mode = iota
	Light
	Dark
	Black
)

// String names the mode, as the appearance button shows it.
func (m Mode) String() string {
	switch m {
	case Light:
		return "浅色"
	case Dark:
		return "深色"
	case Black:
		return "纯黑"
	default:
		return "跟随系统"
	}
}

// Key names the mode as the settings file keeps it.
func (m Mode) Key() string {
	switch m {
	case Light:
		return "light"
	case Dark:
		return "dark"
	case Black:
		return "black"
	default:
		return "system"
	}
}

// ParseMode reads the name a settings file keeps; anything else follows the
// desktop.
func ParseMode(key string) Mode {
	switch key {
	case "light":
		return Light
	case "dark":
		return Dark
	case "black":
		return Black
	default:
		return System
	}
}

// Next returns the mode the appearance button cycles to.
func (m Mode) Next() Mode { return (m + 1) % 4 }
