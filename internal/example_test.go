package internal

import "testing"

func TestAbs(t *testing.T) {
	got := Square(-1, -1)
	if got != 1 {
		t.Errorf("Abs(-1) = %d; want 1", got)
	}
}
