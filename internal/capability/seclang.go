package capability

import "strings"

// NextToken returns the next SecLang token and the remaining input. Quoted
// tokens preserve backslash escapes exactly so callers can make conservative
// semantic decisions without inventing a second tokenizer.
func NextToken(s string) (string, string, bool) {
	if s == "" {
		return "", "", false
	}
	if s[0] == '"' || s[0] == '\'' {
		q := s[0]
		var b strings.Builder
		escaped := false
		for i := 1; i < len(s); i++ {
			c := s[i]
			if escaped {
				b.WriteByte(c)
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				b.WriteByte(c)
				continue
			}
			if c == q {
				return b.String(), s[i+1:], true
			}
			b.WriteByte(c)
		}
		return "", "", false
	}
	if i := strings.IndexAny(s, " \t\r\n"); i >= 0 {
		return s[:i], s[i:], true
	}
	return s, "", true
}

// SplitActions separates a SecLang action list without splitting commas inside
// quoted values. It is shared by offline CRS coverage analysis and the live
// VectorScan rule classifier so both paths classify the same source text.
func SplitActions(s string) []string {
	var out []string
	start := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// SplitAction separates an action name from its optional value.
func SplitAction(s string) (string, string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return strings.TrimSpace(s), ""
}

// TrimActionValue removes one matching quote pair after trimming surrounding
// whitespace. It deliberately does not unescape or reinterpret content.
func TrimActionValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}
