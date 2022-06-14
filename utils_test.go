package pacemaker

import "testing"

func TestAtLeast(t *testing.T) {
	atLeast1 := AtLeast(1)

	if 1 != atLeast1(0) {
		t.Errorf("unexpected value, want 1, have 0")
	}
}
