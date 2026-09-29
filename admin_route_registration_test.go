package main

import "testing"

// TestAdminHandlerRouteRegistrationNoPanic protects startup against ambiguous
// net/http ServeMux patterns. A route conflict panics during handler wiring,
// before the admin server can start serving requests.
func TestAdminHandlerRouteRegistrationNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("admin handler route registration panicked: %v", r)
		}
	}()
	_ = (&adminServer{}).handler()
}
