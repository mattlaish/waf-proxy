package vectoraccel

import "testing"

func TestApplyExactTransformsCoraza37Semantics(t *testing.T) {
	tests := []struct {
		name string
		in   string
		ts   []TransformKind
		want string
	}{
		{name: "lowercase", in: "MiXeD-Ä", ts: []TransformKind{TransformLowercase}, want: "mixed-ä"},
		{name: "uppercase", in: "MiXeD-ä", ts: []TransformKind{TransformUppercase}, want: "MIXED-Ä"},
		{name: "trim", in: " \t\nfoo\r\f\v", ts: []TransformKind{TransformTrim}, want: "foo"},
		{name: "trim-left", in: " \tfoo ", ts: []TransformKind{TransformTrimLeft}, want: "foo "},
		{name: "trim-right", in: " foo\t ", ts: []TransformKind{TransformTrimRight}, want: " foo"},
		{name: "remove-nulls", in: "fo\x00o", ts: []TransformKind{TransformRemoveNulls}, want: "foo"},
		{name: "replace-nulls", in: "fo\x00o", ts: []TransformKind{TransformReplaceNulls}, want: "fo o"},
		{name: "compress-whitespace", in: "a\t \r\n\vb", ts: []TransformKind{TransformCompressWhitespace}, want: "a b"},
		{name: "compress-latin1-whitespace", in: string([]byte{'a', 0x85, 0xA0, 'b'}), ts: []TransformKind{TransformCompressWhitespace}, want: "a b"},
		{name: "remove-whitespace", in: "a \t\n\u00a0b", ts: []TransformKind{TransformRemoveWhitespace}, want: "ab"},
		{name: "length-is-bytes", in: "é", ts: []TransformKind{TransformLength}, want: "2"},
		{name: "base64-encode", in: "abc\x00", ts: []TransformKind{TransformBase64Encode}, want: "YWJjAA=="},
		{name: "hex-encode", in: "é", ts: []TransformKind{TransformHexEncode}, want: "c3a9"},
		{name: "order", in: "  MiXeD  ", ts: []TransformKind{TransformTrim, TransformUppercase}, want: "MIXED"},
		{name: "encoding-pipeline", in: " abc ", ts: []TransformKind{TransformTrim, TransformBase64Encode, TransformHexEncode}, want: "59574a6a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := applyExactTransforms(tc.in, tc.ts); got != tc.want {
				t.Fatalf("applyExactTransforms(%q, %v)=%q want %q", tc.in, tc.ts, got, tc.want)
			}
		})
	}
}

func TestParseExactTransformAllowList(t *testing.T) {
	allowed := []string{
		"lowercase", "uppercase", "trim", "trimLeft", "trimRight",
		"removeNulls", "replaceNulls", "compressWhitespace", "removeWhitespace", "length",
		"base64Encode", "hexEncode",
	}
	for _, name := range allowed {
		if _, ok := parseExactTransform(name); !ok {
			t.Fatalf("expected %q to be allowed", name)
		}
	}
	for _, name := range []string{"urlDecode", "urlDecodeUni", "normalizePath", "htmlEntityDecode", "jsDecode", "cmdLine", "base64Decode", "hexDecode", "md5", "sha1"} {
		if _, ok := parseExactTransform(name); ok {
			t.Fatalf("unsafe/unqualified transform %q unexpectedly allowed", name)
		}
	}
}
