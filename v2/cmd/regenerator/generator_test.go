package main

import (
	"testing"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/testutil"
)

func TestGenerateWrapper(t *testing.T) {
	wrapper, err := generateWrapper("v2.0.0")
	if err != nil {
		t.Fatal(err)
	}

	testutil.AssertGolden(t, "test-fixtures/buildutilw-golden.sh", []byte(wrapper))
}
