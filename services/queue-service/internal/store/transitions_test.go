package store

import "testing"

func TestValidTransition(t *testing.T) {
	cases := []struct {
		action string
		from   string
		valid  bool
	}{
		{"call_next", "waiting", true},
		{"call_next", "serving", false},
		{"start_serving", "called", true},
		{"start_serving", "waiting", false},
		{"complete", "serving", true},
		{"complete", "called", false},
		{"cancel", "waiting", true},
		{"cancel", "called", false},
		{"hold", "waiting", true},
		{"hold", "serving", false},
		{"unhold", "held", true},
		{"unhold", "waiting", false},
		{"recall", "called", true},
		{"recall", "done", false},
		{"transfer", "waiting", true},
		{"transfer", "serving", true},
		{"transfer", "done", false},
		{"no_show", "called", true},
		{"no_show", "waiting", false},
		{"unknown", "waiting", false},
	}

	for _, tt := range cases {
		if got := ValidTransition(tt.action, tt.from); got != tt.valid {
			t.Fatalf("ValidTransition(%q, %q)=%v, want %v", tt.action, tt.from, got, tt.valid)
		}
	}
}
