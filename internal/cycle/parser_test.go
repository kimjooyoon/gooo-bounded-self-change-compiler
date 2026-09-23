package cycle

import "testing"

func TestKeyValuesRejectsDuplicateKeys(t *testing.T) {
	if _, err := keyValues([]string{"frontier=first", "frontier=second"}); err == nil {
		t.Fatal("duplicate cycle declaration key was accepted")
	}
}
