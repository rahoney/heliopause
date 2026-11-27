package cli

// A decoder keeps at most one short terminal sequence. Overlong CSI input is
// discarded until its terminator; its tail must not become an Enter/q action.
type launcherKeyDecoder struct {
	sequence [32]byte
	length   int
	dropping bool
}

func (d *launcherKeyDecoder) pending() bool { return d.length > 0 }

func (d *launcherKeyDecoder) expire() launcherKey {
	key := launcherIdle
	if d.length == 1 {
		key = launcherBack
	}
	d.length, d.dropping = 0, false
	return key
}

func (d *launcherKeyDecoder) feed(value byte) (launcherKey, bool) {
	if d.length == 0 {
		switch value {
		case 27:
			d.sequence[0], d.length = value, 1
			return launcherIdle, false
		case 3:
			return launcherInterrupt, true
		case 4, 'q', 'Q':
			return launcherQuit, true
		case '\r', '\n':
			return launcherSelect, true
		case 8, 127:
			return launcherBack, true
		}
		return launcherIdle, false
	}
	if d.length == 1 && value != '[' && value != 'O' {
		d.length = 0
		return launcherBack, true
	}
	if d.length == 1 {
		d.sequence[1], d.length = value, 2
		return launcherIdle, false
	}
	final := value >= 0x40 && value <= 0x7e
	if d.dropping || d.length == len(d.sequence) {
		d.dropping = true
		if final {
			d.length, d.dropping = 0, false
		}
		return launcherIdle, false
	}
	d.sequence[d.length] = value
	d.length++
	if !final {
		return launcherIdle, false
	}
	sequence := string(d.sequence[2:d.length])
	d.length = 0
	switch sequence {
	case "A":
		return launcherUp, true
	case "B":
		return launcherDown, true
	case "C":
		return launcherSelect, true
	case "D":
		return launcherBack, true
	case "H", "1~", "7~":
		return launcherFirst, true
	case "F", "4~", "8~":
		return launcherLast, true
	case "5~":
		return launcherPageUp, true
	case "6~":
		return launcherPageDown, true
	}
	return launcherIdle, false
}
