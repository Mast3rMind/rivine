package build

import (
	"fmt"
	"testing"
)

// TestVersionCmp checks that in all cases, VersionCmp returns the correct
// result.
func TestVersionCmp(t *testing.T) {
	versionTests := []struct {
		a, b ProtocolVersion
		exp  int
	}{
		{NewVersion(0, 1, 0), NewVersion(0, 0, 9), 1},
		{NewVersion(0, 1, 0), NewVersion(0, 1, 0), 0},
		{NewVersion(0, 1, 0), NewVersion(0, 1, 1), -1},
		{NewVersion(0, 1, 0), NewVersion(1, 1, 0), -1},
		{NewPrereleaseVersion(0, 1, 1, "0"), NewVersion(0, 1, 1), -1},
	}

	for _, test := range versionTests {
		if actual := test.a.Compare(test.b); actual != test.exp {
			t.Errorf("Comparing %s to %s should return %v (got %v)",
				test.a.String(), test.b.String(), test.exp, actual)
		}
	}
}

func TestVersionStringReflection(t *testing.T) {
	testCases := []struct {
		in, out string
	}{
		{"1", "1.0.0"},
		{"1.1", "1.1.0"},
		{"1.1.1", "1.1.1"},
		{"1.1.1-1", "1.1.1-1\x00\x00\x00\x00\x00\x00\x00"},
		{"255.255.255-12345678", "255.255.255-12345678"},
		{"000.000.000-00000000", "0.0.0-00000000"},
		{"1.2.3-alpha", "1.2.3-alpha\x00\x00\x00"},
		{"1-4", "1.0.0-4\x00\x00\x00\x00\x00\x00\x00"},
		{"1.2-4", "1.2.0-4\x00\x00\x00\x00\x00\x00\x00"},
		{"1.2.3-4", "1.2.3-4\x00\x00\x00\x00\x00\x00\x00"},
	}

	for index, testCase := range testCases {
		// pass 1
		version, err := Parse(testCase.in)
		if err != nil {
			t.Errorf("test %d (pass 1) failed: %v", index, err)
			continue
		}
		out := version.String()
		if testCase.out != out {
			t.Errorf("test %d (pass 1) failed: expected %q, while received %q", index, testCase.out, out)
			continue
		}
		// pass 2
		version2, err := Parse("v" + testCase.in)
		if err != nil {
			t.Errorf("test %d (pass 2) failed: %v", index, err)
			continue
		}
		if version.Compare(version2) != 0 {
			t.Errorf("test %d (pass 2) failed: expected %q, while received %q", index, version, version2)
			continue
		}
		if out2 := version2.String(); out != out2 {
			t.Errorf("test %d (pass 2) failed: expected %q, while received %q", index, out, out2)
		}
	}
}

func TestVersionJSONReflection(t *testing.T) {
	testCases := []ProtocolVersion{
		NewVersion(0, 0, 0),
		NewVersion(1, 2, 3),
		NewPrereleaseVersion(1, 2, 3, "4"),
		NewPrereleaseVersion(255, 255, 255, "        "),
	}
	for index, in := range testCases {
		bytes, err := in.MarshalJSON()
		if err != nil {
			t.Errorf("test %d failed: MarshalJSON: %v", index, err)
			continue
		}

		var out ProtocolVersion
		err = out.UnmarshalJSON(bytes)
		if err != nil {
			t.Errorf("test %d failed: UnmarshalJSON: %v", index, err)
			continue
		}

		if in.String() != out.String() {
			t.Errorf("test %d failed: expected %q, while received %q", index, in, out)
		}
	}
}

func TestInvalidStringVersionRange(t *testing.T) {
	if _, err := Parse("256"); err == nil {
		t.Fatal("expected `256` to be out of range")
	}
	if _, err := Parse("1.256"); err == nil {
		t.Fatal("expected `1.256` to be out of range")
	}
	if _, err := Parse("1.1.256"); err == nil {
		t.Fatal("expected `1.1.256` to be out of range")
	}
	if _, err := Parse("1.256.256"); err == nil {
		t.Fatal("expected `1.256.256` to be out of range")
	}
	if _, err := Parse("256.256.256"); err == nil {
		t.Fatal("expected `256.256.256` to be out of range")
	}
}

func TestValidStringVersionRange(t *testing.T) {
	for major := 0; major <= 255; major += 25 {
		for minor := 0; minor <= 255; minor += 15 {
			for patch := 0; patch <= 255; patch++ {
				raw := fmt.Sprintf("%d.%d.%d", major, minor, patch)
				version, err := Parse(raw)
				if err != nil {
					t.Errorf("test %q failed: %v", raw, err)
					continue
				}
				if out := version.String(); raw != out {
					t.Errorf("test failed: expected %q, while received %q", raw, out)
				}
			}
		}
	}
}
