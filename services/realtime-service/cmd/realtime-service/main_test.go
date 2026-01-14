package main

import "testing"

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		branchID  string
		serviceID string
		branches  []string
		services  []string
		want      bool
	}{
		{"no restrictions", "b1", "s1", nil, nil, true},
		{"branch allowed", "b1", "s1", []string{"b1"}, nil, true},
		{"branch denied", "b2", "s1", []string{"b1"}, nil, false},
		{"service allowed", "b1", "s1", nil, []string{"s1"}, true},
		{"service denied", "b1", "s2", nil, []string{"s1"}, false},
		{"both allowed", "b1", "s1", []string{"b1"}, []string{"s1"}, true},
		{"missing branch", "", "s1", []string{"b1"}, nil, false},
		{"missing service", "b1", "", nil, []string{"s1"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAllowed(tc.branchID, tc.serviceID, tc.branches, tc.services); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
