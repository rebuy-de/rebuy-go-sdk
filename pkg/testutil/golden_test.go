package testutil_test

import (
	"testing"

	"github.com/rebuy-de/rebuy-go-sdk/v5/pkg/testutil"
)

type exampleData struct {
	Foo     string `json:"foo"`
	Bim     string `json:"bim"`
	Blubber int    `json:"blubber"`
}

func TestAssertGoldenJSON(t *testing.T) {
	data := exampleData{
		Foo:     "bar",
		Bim:     "baz",
		Blubber: 42,
	}

	testutil.AssertGoldenJSON(t, "test-fixtures/example-golden.json", data)
}

func TestAssertGoldenDiff(t *testing.T) {
	lhs := exampleData{
		Foo:     "bar",
		Bim:     "baz",
		Blubber: 42,
	}

	rhs := exampleData{
		Foo:     "bir",
		Bim:     "baz",
		Blubber: 177,
	}

	testutil.AssertGoldenDiffJSON(t, "test-fixtures/example-golden.diff", lhs, rhs)
}
