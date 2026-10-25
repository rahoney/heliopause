package npm

import (
	"bytes"
	"errors"
	"unicode/utf8"
)

var errBunLock = errors.New("bun public lock selection is invalid or unsupported")

// Bun writes JSON with trailing commas. Comments are allowed only outside
// strings. Keep byte positions and string bytes while removing that syntax;
// the existing strict JSON reader still owns structure, duplicates and budgets.
func normalizeBunLockJSON(body []byte) ([]byte, error) {
	if len(body) == 0 || len(body) > ProjectControlLimit || !utf8.Valid(body) {
		return nil, errBunLock
	}
	output := bytes.Clone(body)
	for i := 0; i < len(output); {
		switch {
		case output[i] == '"':
			end, err := bunJSONStringEnd(output, i)
			if err != nil {
				return nil, err
			}
			i = end
		case output[i] == '/' && i+1 < len(output) && output[i+1] == '/':
			for i < len(output) && output[i] != '\n' && output[i] != '\r' {
				output[i] = ' '
				i++
			}
		case output[i] == '/' && i+1 < len(output) && output[i+1] == '*':
			output[i], output[i+1] = ' ', ' '
			i += 2
			closed := false
			for i < len(output) {
				if output[i] == '*' && i+1 < len(output) && output[i+1] == '/' {
					output[i], output[i+1] = ' ', ' '
					i += 2
					closed = true
					break
				}
				if output[i] != '\n' && output[i] != '\r' {
					output[i] = ' '
				}
				i++
			}
			if !closed {
				return nil, errBunLock
			}
		default:
			i++
		}
	}
	previous := byte(0)
	for i := 0; i < len(output); {
		if bunJSONSpace(output[i]) {
			i++
			continue
		}
		if output[i] == '"' {
			end, err := bunJSONStringEnd(output, i)
			if err != nil {
				return nil, err
			}
			previous, i = '"', end
			continue
		}
		if output[i] == ',' {
			next := i + 1
			for next < len(output) && bunJSONSpace(output[next]) {
				next++
			}
			if next < len(output) && (output[next] == '}' || output[next] == ']') {
				// Do not repair {,}, [,], missing values or doubled commas.
				if previous != '"' && previous != '}' && previous != ']' &&
					previous != 'e' && previous != 'l' && (previous < '0' || previous > '9') {
					return nil, errBunLock
				}
				output[i] = ' '
			}
		}
		previous = output[i]
		i++
	}
	if strictControlJSON(output) != nil {
		return nil, errBunLock
	}
	return output, nil
}

func bunJSONSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

// Reject unpaired escaped UTF-16 before encoding/json could replace it with
// U+FFFD. Otherwise distinct untrusted names could acquire the same spelling.
func bunJSONStringEnd(body []byte, start int) (int, error) {
	for i := start + 1; i < len(body); i++ {
		if body[i] == '"' {
			return i + 1, nil
		}
		if body[i] < 0x20 {
			return 0, errBunLock
		}
		if body[i] != '\\' {
			continue
		}
		i++
		if i >= len(body) {
			return 0, errBunLock
		}
		switch body[i] {
		case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		case 'u':
			unit, ok := bunJSONHexUnit(body, i+1)
			if !ok || unit >= 0xdc00 && unit <= 0xdfff {
				return 0, errBunLock
			}
			i += 4
			if unit >= 0xd800 && unit <= 0xdbff {
				if i+2 >= len(body) || body[i+1] != '\\' || body[i+2] != 'u' {
					return 0, errBunLock
				}
				low, ok := bunJSONHexUnit(body, i+3)
				if !ok || low < 0xdc00 || low > 0xdfff {
					return 0, errBunLock
				}
				i += 6
			}
		default:
			return 0, errBunLock
		}
	}
	return 0, errBunLock
}

func bunJSONHexUnit(body []byte, start int) (uint16, bool) {
	if start > len(body)-4 {
		return 0, false
	}
	var value uint16
	for _, b := range body[start : start+4] {
		value <<= 4
		switch {
		case b >= '0' && b <= '9':
			value += uint16(b - '0')
		case b >= 'a' && b <= 'f':
			value += uint16(b-'a') + 10
		case b >= 'A' && b <= 'F':
			value += uint16(b-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}
