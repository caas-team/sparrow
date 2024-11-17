package test

import "testing"

// MarkAsShort marks the test as short, so it will be skipped if the -test.short flag is not provided.
func MarkAsShort(t *testing.T) {
	t.Helper()
	if !testing.Short() {
		t.Skip("skipping short tests, to run them use the -test.short flag")
	}
}

// MarkAsLong marks the test as long, so it will be skipped if the -test.short flag is provided.
func MarkAsLong(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping long tests, to run them remove the -test.short flag")
	}
}
