package capability

import "testing"

func TestSecLangTokenAndActionHelpers(t *testing.T) {
	tok, rest, ok := NextToken(`"ARGS:id" "@rx ^[0-9]+$" "id:1,phase:2,t:none,msg:'a,b'"`)
	if !ok || tok != "ARGS:id" || rest == "" {
		t.Fatalf("unexpected first token: %q %q %v", tok, rest, ok)
	}
	actions := SplitActions(`id:1,phase:2,msg:'a,b',t:"lowercase"`)
	if len(actions) != 4 {
		t.Fatalf("actions=%#v", actions)
	}
	name, value := SplitAction(actions[2])
	if name != "msg" || TrimActionValue(value) != "a,b" {
		t.Fatalf("action=%q value=%q", name, value)
	}
	name, value = SplitAction("chain")
	if name != "chain" || value != "" {
		t.Fatalf("flag action=%q value=%q", name, value)
	}
}
