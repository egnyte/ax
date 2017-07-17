package kibana

import "testing"

func TestSafeFilename(t *testing.T) {
	if safeFilename("turbo-*") != "turbo-_" {
		t.Fatal("Not a safe filename:", safeFilename("turbo-*"))
	}
}
