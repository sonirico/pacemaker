package main

import "testing"

func assertFreeSlots(t *testing.T, expected, actual int64) {
	t.Helper()
	if expected != actual {
		t.Errorf("unexpected free slots, want %d, have %d", expected, actual)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error, want none, have %v", err)
	}
}
