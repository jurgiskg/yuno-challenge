package country

import "testing"

func TestCode(t *testing.T) {
	tests := []struct {
		code      Code
		wantValid bool
		wantName  string
	}{
		{MX, true, "Mexico"},
		{CO, true, "Colombia"},
		{BR, true, "Brazil"},
		{CL, true, "Chile"},
		{"XX", false, ""},
		{"mx", false, ""},
		{"", false, ""},
	}
	for _, tt := range tests {
		if got := tt.code.Valid(); got != tt.wantValid {
			t.Errorf("%q.Valid() = %v, want %v", tt.code, got, tt.wantValid)
		}
		if got := tt.code.Name(); got != tt.wantName {
			t.Errorf("%q.Name() = %q, want %q", tt.code, got, tt.wantName)
		}
	}
	if len(names) != 249 {
		t.Errorf("got %d codes, want the 249 officially assigned", len(names))
	}
}
