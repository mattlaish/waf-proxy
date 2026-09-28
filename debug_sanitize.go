package main

import "strings"

func isSensitiveDebugKey(k string) bool {
	k = strings.ToLower(strings.TrimSpace(k))
	for _, part := range []string{"authorization", "proxy-authorization", "cookie", "set-cookie", "password", "passwd", "secret", "api_key", "apikey", "token", "private_key", "peer_token"} {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

func sanitizeDebugValue(key string, v any) any {
	if isSensitiveDebugKey(key) {
		return "[masked]"
	}
	switch x := v.(type) {
	case string:
		return truncateDebugString(x, 4096)
	case []string:
		out := make([]string, len(x))
		for i := range x {
			out[i] = truncateDebugString(x[i], 4096)
		}
		return out
	case map[string]any:
		return sanitizeDebugMap(x)
	case map[string]string:
		out := make(map[string]any, len(x))
		for k, v := range x {
			out[k] = sanitizeDebugValue(k, v)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = sanitizeDebugValue("", x[i])
		}
		return out
	default:
		return v
	}
}

func sanitizeDebugMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = sanitizeDebugValue(k, v)
	}
	return out
}
