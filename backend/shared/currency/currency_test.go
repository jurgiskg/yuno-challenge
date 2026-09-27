package currency

import "testing"

func TestCode(t *testing.T) {
	tests := []struct {
		code         Code
		wantValid    bool
		wantName     string
		wantDecimals int32
	}{
		{MXN, true, "Mexican Peso", 2},
		{COP, true, "Colombian Peso", 2},
		{BRL, true, "Brazilian Real", 2},
		{CLP, true, "Chilean Peso", 0},
		{KWD, true, "Kuwaiti Dinar", 3},
		{"XXX", false, "", 0},
		{"mxn", false, "", 0},
		{"", false, "", 0},
	}
	for _, tt := range tests {
		if got := tt.code.Valid(); got != tt.wantValid {
			t.Errorf("%q.Valid() = %v, want %v", tt.code, got, tt.wantValid)
		}
		if got := tt.code.Name(); got != tt.wantName {
			t.Errorf("%q.Name() = %q, want %q", tt.code, got, tt.wantName)
		}
		if got := tt.code.Decimals(); got != tt.wantDecimals {
			t.Errorf("%q.Decimals() = %d, want %d", tt.code, got, tt.wantDecimals)
		}
	}
	if len(currencies) != 155 {
		t.Errorf("got %d codes, want the 155 active currencies", len(currencies))
	}
}
