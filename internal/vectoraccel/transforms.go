package vectoraccel

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode"
)

// TransformKind is a deliberately small allow-list of Coraza v3.7.0
// transformations whose input/output semantics are reproduced locally without
// parsing or transaction state. Unsupported transforms keep the rule Coraza-only.
type TransformKind string

const (
	TransformLowercase          TransformKind = "lowercase"
	TransformUppercase          TransformKind = "uppercase"
	TransformTrim               TransformKind = "trim"
	TransformTrimLeft           TransformKind = "trimleft"
	TransformTrimRight          TransformKind = "trimright"
	TransformRemoveNulls        TransformKind = "removenulls"
	TransformReplaceNulls       TransformKind = "replacenulls"
	TransformCompressWhitespace TransformKind = "compresswhitespace"
	TransformRemoveWhitespace   TransformKind = "removewhitespace"
	TransformLength             TransformKind = "length"
	TransformBase64Encode       TransformKind = "base64encode"
	TransformHexEncode          TransformKind = "hexencode"
)

const corazaTrimSpaces = " \t\n\r\f\v"

func parseExactTransform(name string) (TransformKind, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case string(TransformLowercase):
		return TransformLowercase, true
	case string(TransformUppercase):
		return TransformUppercase, true
	case string(TransformTrim):
		return TransformTrim, true
	case string(TransformTrimLeft):
		return TransformTrimLeft, true
	case string(TransformTrimRight):
		return TransformTrimRight, true
	case string(TransformRemoveNulls):
		return TransformRemoveNulls, true
	case string(TransformReplaceNulls):
		return TransformReplaceNulls, true
	case string(TransformCompressWhitespace):
		return TransformCompressWhitespace, true
	case string(TransformRemoveWhitespace):
		return TransformRemoveWhitespace, true
	case string(TransformLength):
		return TransformLength, true
	case string(TransformBase64Encode):
		return TransformBase64Encode, true
	case string(TransformHexEncode):
		return TransformHexEncode, true
	default:
		return "", false
	}
}

func transformKey(ts []TransformKind) string {
	if len(ts) == 0 {
		return "none"
	}
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = string(t)
	}
	return strings.Join(parts, ">")
}

func applyExactTransforms(value string, ts []TransformKind) string {
	for _, t := range ts {
		switch t {
		case TransformLowercase:
			value = strings.ToLower(value)
		case TransformUppercase:
			value = strings.ToUpper(value)
		case TransformTrim:
			value = strings.Trim(value, corazaTrimSpaces)
		case TransformTrimLeft:
			value = strings.TrimLeft(value, corazaTrimSpaces)
		case TransformTrimRight:
			value = strings.TrimRight(value, corazaTrimSpaces)
		case TransformRemoveNulls:
			value = strings.ReplaceAll(value, "\x00", "")
		case TransformReplaceNulls:
			value = strings.ReplaceAll(value, "\x00", " ")
		case TransformCompressWhitespace:
			value = compressCorazaWhitespace(value)
		case TransformRemoveWhitespace:
			value = strings.Map(func(r rune) rune {
				if unicode.IsSpace(r) {
					return -1
				}
				return r
			}, value)
		case TransformLength:
			value = strconv.Itoa(len(value))
		case TransformBase64Encode:
			value = base64.StdEncoding.EncodeToString([]byte(value))
		case TransformHexEncode:
			value = hex.EncodeToString([]byte(value))
		}
	}
	return value
}

func isCorazaLatinSpace(c byte) bool {
	switch c {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	default:
		return false
	}
}

func compressCorazaWhitespace(value string) string {
	first := -1
	for i := 0; i < len(value); i++ {
		if isCorazaLatinSpace(value[i]) {
			first = i
			break
		}
	}
	if first < 0 {
		return value
	}
	out := make([]byte, 0, len(value))
	out = append(out, value[:first]...)
	inWhitespace := false
	for i := first; i < len(value); i++ {
		if isCorazaLatinSpace(value[i]) {
			if !inWhitespace {
				out = append(out, ' ')
				inWhitespace = true
			}
			continue
		}
		inWhitespace = false
		out = append(out, value[i])
	}
	return string(out)
}
