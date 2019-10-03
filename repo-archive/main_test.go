package main

import (
	"os"
	"testing"
)

func TestDateStr(t *testing.T) {
	if os.Getenv("hack") == "" {
		t.Skip("hack")
	}

	t.Log("dateStr():", dateStr())
	if ds := dateStr(); len(dateStr()) != 6 {
		t.Errorf("unexpected dateStr: %v", ds)
	}
}
