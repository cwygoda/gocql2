package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var cqlNumberPattern = regexp.MustCompile(`^[+-]?(?:(?:[0-9]+(?:\.[0-9]*)?)|(?:\.[0-9]+))(?:[eE][+-]?[0-9]+)?$`)

func canonicalNumber(raw string) (string, error) {
	if !cqlNumberPattern.MatchString(raw) {
		return "", fmt.Errorf("invalid numeric literal %q", raw)
	}

	s := raw
	negative := false
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		negative = s[0] == '-'
		s = s[1:]
	}

	exp := 0
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		var err error
		expPart := s[i+1:]
		exp, err = parseSmallInt(expPart)
		if err != nil {
			return "", fmt.Errorf("invalid numeric exponent %q", expPart)
		}
		s = s[:i]
	}

	intPart, fracPart := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if intPart == "" {
		intPart = "0"
	}

	digits := strings.TrimLeft(intPart+fracPart, "0")
	if digits == "" {
		return "0", nil
	}
	scale := len(fracPart) - exp

	var out string
	if scale <= 0 {
		out = digits + strings.Repeat("0", -scale)
	} else if scale >= len(digits) {
		out = "0." + strings.Repeat("0", scale-len(digits)) + digits
	} else {
		point := len(digits) - scale
		out = digits[:point] + "." + digits[point:]
	}

	if strings.Contains(out, ".") {
		out = strings.TrimRight(out, "0")
		out = strings.TrimRight(out, ".")
	}
	if negative {
		out = "-" + out
	}
	return out, nil
}

func parseSmallInt(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty integer")
	}
	sign := 1
	if s[0] == '+' || s[0] == '-' {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	if s == "" {
		return 0, fmt.Errorf("empty integer")
	}
	value := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		value = value*10 + int(r-'0')
		if value > 1_000_000 {
			return 0, fmt.Errorf("exponent too large")
		}
	}
	return sign * value, nil
}
