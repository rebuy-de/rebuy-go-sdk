package testutil

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

var (
	updateGolden = flag.Bool("update-golden", false,
		"update the golden file instead of comparing it")
)

// AssertGolden tests, if the content of filename matches given data. On
// missmatch the test fails. When setting the `-update-golden` flag to the
// test, if will update the file which can be compared via a VCS diff.
func AssertGolden(t *testing.T, filename string, data []byte) {
	if *updateGolden {
		err := ioutil.WriteFile(filename, data, os.FileMode(0644))
		if err != nil {
			t.Error(err)
			return
		}
	}

	golden, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Error(err)
		return
	}

	if string(golden) != string(data) {
		t.Errorf("Generated file '%s' doesn't match golden file. Update with '-update-golden'.", filename)
	}
}

// AssertGoldenYAML works like AssertGolden, but converts the data to YAML file.
func AssertGoldenYAML(t *testing.T, filename string, data interface{}) {
	generated, err := yaml.Marshal(data)
	if err != nil {
		t.Error(err)
		return
	}

	generated = append(generated, '\n')

	AssertGolden(t, filename, generated)
}

// AssertGoldenJSON works like AssertGolden, but converts the data to JSON file.
func AssertGoldenJSON(t *testing.T, filename string, data interface{}) {
	generated, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		t.Error(err)
		return
	}

	generated = append(generated, '\n')

	AssertGolden(t, filename, generated)
}
