package testutil

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pmezard/go-difflib/difflib"
	yaml "gopkg.in/yaml.v3"
)

const GoldenUpdateEnv = `TESTUTIL_UPDATE_GOLDEN`

// TB is a interface that is a subset of the testing.TB interface and therefore
// every *testing.T struct can be used.
type TB interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Log(args ...interface{})
}

func assertGolden(t TB, filename string, data []byte, showDiff bool) {
	if os.Getenv(GoldenUpdateEnv) != "" {
		err := ioutil.WriteFile(filename, data, os.FileMode(0644))
		if err != nil {
			t.Error(err)
			return
		}
	}

	golden, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		golden = []byte{}
	} else if err != nil {
		t.Error(err)
		return
	}

	udiff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(golden)),
		B:        difflib.SplitLines(string(data)),
		FromFile: filename,
		ToFile:   "Current",
		Context:  3,
		Eol:      "\n",
	})

	if err != nil {
		t.Error(err)
		return
	}

	if udiff != "" {
		t.Errorf("Generated file '%s' doesn't match golden file. Update it by setting the environment variable %s.", filename, GoldenUpdateEnv)
		if showDiff {
			t.Log(udiff)
		}
	}
}

// AssertGolden tests, if the content of filename matches given data. On
// missmatch the test fails. When setting the TESTUTIL_UPDATE_GOLDEN
// environment variable, it will update the file which can be compared via a
// VCS diff.
func AssertGolden(t TB, filename string, data []byte) {
	assertGolden(t, filename, data, true)
}

// AssertGoldenYAML works like AssertGolden, but converts the data to YAML file.
func AssertGoldenYAML(t TB, filename string, data interface{}) {
	generated, err := yaml.Marshal(data)
	if err != nil {
		t.Error(err)
		return
	}

	generated = append(generated, '\n')

	AssertGolden(t, filename, generated)
}

// AssertGoldenJSON works like AssertGolden, but converts the data to JSON file.
func AssertGoldenJSON(t TB, filename string, data interface{}) {
	generated, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		t.Error(err)
		return
	}

	generated = append(generated, '\n')

	AssertGolden(t, filename, generated)
}

// AssertGoldenDiff creates a unified diff of two texts and compares it with the golden file.
func AssertGoldenDiff(t TB, filename string, a, b string) {
	udiff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(a),
		B:        difflib.SplitLines(b),
		FromFile: "a",
		ToFile:   "b",
		Context:  3,
		Eol:      "\n",
	})

	if err != nil {
		t.Error(err)
		return
	}

	assertGolden(t, filename, []byte(udiff), false)
}

// AssertGoldenDiffJSON works like AssertGoldenDiff, but converts the data to JSON first.
func AssertGoldenDiffJSON(t TB, filename string, a, b interface{}) {
	aJSON, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		t.Error(err)
		return
	}

	bJSON, err := json.MarshalIndent(b, "", "    ")
	if err != nil {
		t.Error(err)
		return
	}

	AssertGoldenDiff(t, filename, string(aJSON), string(bJSON))
}
