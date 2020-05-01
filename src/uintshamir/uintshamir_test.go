package uintshamir

import "testing"

func TestMod(t *testing.T) {
	aModB := mod(3, -7)
	if aModB != 3 {
		t.Errorf("asd")
	}
}
