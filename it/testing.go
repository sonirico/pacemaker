package main

import (
	"errors"
	"testing"
)

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

func assertError(t *testing.T, expected, actual error) {
	t.Helper()

	if actual == nil {
		t.Errorf("expected error, have none")
	}

	if !errors.Is(expected, actual) {
		t.Errorf("expected type of %v, have %v", expected, actual)
	}
}
