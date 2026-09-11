//go:build realcoraza

package vectoraccel

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/corazawaf/coraza/v3"
)

func realCorazaMatchedRule(t *testing.T, variable, pattern, transforms, value string) bool {
	t.Helper()
	directives := fmt.Sprintf(`
SecRuleEngine DetectionOnly
SecRule %s "@rx %s" "id:188001,phase:1,pass,nolog,t:none%s"
`, variable, pattern, transforms)
	waf, err := coraza.NewWAF(coraza.NewWAFConfig().WithDirectives(directives))
	if err != nil {
		t.Fatalf("NewWAF(%s): %v", variable, err)
	}
	tx := waf.NewTransaction()
	defer tx.Close()
	tx.ProcessConnection("127.0.0.1", 12345, "127.0.0.1", 443)
	tx.ProcessURI("/", "GET", "HTTP/1.1")
	tx.AddRequestHeader("X-Vector-Test", value)
	if it := tx.ProcessRequestHeaders(); it != nil {
		t.Fatalf("unexpected interruption: %+v", it)
	}
	tx.ProcessLogging()
	for _, mr := range tx.MatchedRules() {
		if mr.Rule().ID() == 188001 {
			return true
		}
	}
	return false
}

func TestPhase2ExactTransformsAgainstRealCoraza37(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		transforms []TransformKind
		actions    string
	}{
		{name: "lowercase", input: "MiXeD-Ä", transforms: []TransformKind{TransformLowercase}, actions: ",t:lowercase"},
		{name: "uppercase", input: "MiXeD-ä", transforms: []TransformKind{TransformUppercase}, actions: ",t:uppercase"},
		{name: "trim", input: " \tfoo\r\n", transforms: []TransformKind{TransformTrim}, actions: ",t:trim"},
		{name: "trim-left", input: " \tfoo ", transforms: []TransformKind{TransformTrimLeft}, actions: ",t:trimLeft"},
		{name: "trim-right", input: " foo\t ", transforms: []TransformKind{TransformTrimRight}, actions: ",t:trimRight"},
		{name: "remove-nulls", input: "fo\x00o", transforms: []TransformKind{TransformRemoveNulls}, actions: ",t:removeNulls"},
		{name: "replace-nulls", input: "fo\x00o", transforms: []TransformKind{TransformReplaceNulls}, actions: ",t:replaceNulls"},
		{name: "compress-whitespace", input: "a\t \r\nb", transforms: []TransformKind{TransformCompressWhitespace}, actions: ",t:compressWhitespace"},
		{name: "remove-whitespace", input: "a \t\n\u00a0b", transforms: []TransformKind{TransformRemoveWhitespace}, actions: ",t:removeWhitespace"},
		{name: "length", input: "é", transforms: []TransformKind{TransformLength}, actions: ",t:length"},
		{name: "base64-encode", input: "abc\x00", transforms: []TransformKind{TransformBase64Encode}, actions: ",t:base64Encode"},
		{name: "hex-encode", input: "é", transforms: []TransformKind{TransformHexEncode}, actions: ",t:hexEncode"},
		{name: "ordered-pipeline", input: "  MiXeD  ", transforms: []TransformKind{TransformTrim, TransformUppercase}, actions: ",t:trim,t:uppercase"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := applyExactTransforms(tc.input, tc.transforms)
			pattern := "^" + regexp.QuoteMeta(want) + "$"
			if !realCorazaMatchedRule(t, "REQUEST_HEADERS:X-Vector-Test", pattern, tc.actions, tc.input) {
				t.Fatalf("Coraza v3.7.0 did not produce accelerator-equivalent value %q", want)
			}
		})
	}
}

func TestPhase2RequestSourceMappingAgainstRealCoraza37(t *testing.T) {
	waf, err := coraza.NewWAF(coraza.NewWAFConfig().WithDirectives(`
SecRuleEngine DetectionOnly
SecRule QUERY_STRING "@rx ^A=One&B=Two$" "id:188101,phase:1,pass,nolog,t:none"
SecRule SERVER_NAME "@rx ^Public[.]Example:8443$" "id:188102,phase:1,pass,nolog,t:none"
SecRule REQUEST_HEADERS:Host "@rx ^Public[.]Example:8443$" "id:188103,phase:1,pass,nolog,t:none"
SecRule REQUEST_HEADERS:Transfer-Encoding "@rx ^chunked$" "id:188104,phase:1,pass,nolog,t:none"
SecRule REQUEST_URI_RAW "@rx ^/dir/file[.]txt[?]A=One&B=Two$" "id:188105,phase:1,pass,nolog,t:none"
SecRule REQUEST_LINE "@rx ^POST /dir/file[.]txt[?]A=One&B=Two HTTP/2[.]0$" "id:188106,phase:1,pass,nolog,t:none"
SecRule REQUEST_BASENAME "@rx ^file[.]txt$" "id:188107,phase:1,pass,nolog,t:none"
SecRule REMOTE_ADDR "@rx ^203[.]0[.]113[.]9$" "id:188108,phase:1,pass,nolog,t:none"
SecRule REMOTE_PORT "@rx ^45678$" "id:188109,phase:1,pass,nolog,t:none"
`))
	if err != nil {
		t.Fatal(err)
	}
	tx := waf.NewTransaction()
	defer tx.Close()
	tx.ProcessConnection("203.0.113.9", 45678, "127.0.0.1", 443)
	tx.ProcessURI("/dir/file.txt?A=One&B=Two", "POST", "HTTP/2.0")
	tx.AddRequestHeader("Host", "Public.Example:8443")
	tx.SetServerName("Public.Example:8443")
	tx.AddRequestHeader("Transfer-Encoding", "gzip")
	tx.AddRequestHeader("Transfer-Encoding", "chunked")
	if it := tx.ProcessRequestHeaders(); it != nil {
		t.Fatalf("unexpected interruption: %+v", it)
	}
	tx.ProcessLogging()
	got := map[int]bool{}
	for _, mr := range tx.MatchedRules() {
		got[mr.Rule().ID()] = true
	}
	var missing []string
	for _, id := range []int{188101, 188102, 188103, 188104, 188105, 188106, 188107, 188108, 188109} {
		if !got[id] {
			missing = append(missing, fmt.Sprint(id))
		}
	}
	if len(missing) != 0 {
		t.Fatalf("real Coraza source parity missing rule ids: %s", strings.Join(missing, ","))
	}
}
