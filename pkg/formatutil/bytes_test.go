package formatutil

import "testing"

type testCaseByteFormatter struct {
	input    int64
	expected string
}

func TestByteFormatting(t *testing.T) {
	casesIEC := []testCaseByteFormatter{
		{16516, "16.1 KiB"},
		{46564534534, "43.4 GiB"},
		{9845345734653745, "8.7 PiB"},
	}

	for _, testcase := range casesIEC {
		if output := ByteFormatIEC(testcase.input); output != testcase.expected {
			t.Errorf("Output %q not equal to expected %q", output, testcase.expected)
		}
	}

	casesSI := []testCaseByteFormatter{
		{16516, "16.5 kB"},
		{46564534534, "46.6 GB"},
		{9845345734653745, "9.8 PB"},
	}

	for _, testcase := range casesSI {
		if output := ByteFormatSI(testcase.input); output != testcase.expected {
			t.Errorf("Output %q not equal to expected %q", output, testcase.expected)
		}
	}
}
