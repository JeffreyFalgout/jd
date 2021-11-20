package jd

import (
	"testing"
)

func TestDiffAndPatch(t *testing.T) {
	checkDiffAndPatchSuccess(t,
		`{"a":1}`,
		`{"a":2}`,
		`{"a":1,"c":3}`,
		`{"a":2,"c":3}`)
	checkDiffAndPatchSuccess(t,
		`[[]]`,
		`[[1]]`,
		`[[],[2]]`,
		`[[1],[2]]`)
	checkDiffAndPatchSuccess(t,
		`[{"a":1},{"a":1}]`,
		`[{"a":2},{"a":3}]`,
		`[{"a":1},{"a":1,"b":4},{"c":5}]`,
		`[{"a":2},{"a":3,"b":4},{"c":5}]`)
}

func TestAssocIn(t *testing.T) {
	for _, tc := range []struct {
		name string

		// Used to generate the patch.
		oldJSON, newJSON string
		// Apply the patch to this.
		baseJSON string

		want string
	}{
		{
			name:     "single value",
			oldJSON:  `{"foo": {"bar": 1}}`,
			newJSON:  `{"foo": {"bar": 2}}`,
			baseJSON: `{"foo": {"baz": 5}}`,
			want:     `{"foo": {"bar": 2, "baz": 5}}`,
		},
		{
			name:     "objects and values",
			oldJSON:  `{"foo": {"bar": {"baz": 1, "fourth": 9}}}`,
			newJSON:  `{"foo": {"bar": {"baz": 2, "fourth": 10}}}`,
			baseJSON: `{"foo": {"baz": 5}}`,
			want:     `{"foo": {"bar": {"baz": 2, "fourth": 10}, "baz": 5}}`,
		},
		{
			name:     "single array element",
			oldJSON:  `{"foo": []}`,
			newJSON:  `{"foo": [1]}`,
			baseJSON: `{}`,
			want:     `{"foo": [1]}`,
		},
		{
			name:     "suffix array element",
			oldJSON:  `{"foo": [1]}`,
			newJSON:  `{"foo": [1, 2]}`,
			baseJSON: `{}`,
			want:     `{"foo": [1, 2]}`,
		},
		{
			name:     "multiple array elements",
			oldJSON:  `{"foo": [1, 2]}`,
			newJSON:  `{"foo": [1, 3, 4]}`,
			baseJSON: `{}`,
			want:     `{"foo": [1, 3, 4]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldJSON, err := ReadJsonString(tc.oldJSON)
			if err != nil {
				t.Fatalf("ReadJsonString(%q) = %v", tc.oldJSON, err)
			}
			newJSON, err := ReadJsonString(tc.newJSON)
			if err != nil {
				t.Fatalf("ReadJsonString(%q) = %v", tc.newJSON, err)
			}
			baseJSON, err := ReadJsonString(tc.baseJSON)
			if err != nil {
				t.Fatalf("ReadJsonString(%q) = %v", tc.baseJSON, err)
			}
			want, err := ReadJsonString(tc.want)
			if err != nil {
				t.Fatalf("ReadJsonString(%q) = %v", tc.want, err)
			}

			diff := oldJSON.Diff(newJSON, ASSOC_IN)
			got, err := baseJSON.Patch(diff)
			if err != nil {
				t.Errorf("baseJSON.Patch(%q) failed with %v, wanted to be patched successfully resulting in %s", diff.Render(), err, tc.want)
			} else if !want.Equals(got) {
				t.Errorf("baseJSON.Patch(%q) = %v, wanted %v", diff.Render(), got, want)
			}
		})
	}
}

func TestDiffAndPatchSet(t *testing.T) {
	checkDiffAndPatchSuccessSet(t,
		`{"a":{"b" : ["3", "4" ],"c" : ["2", "1"]}}`,
		`{"a":{"b" : ["3", "4", "5", "6"],"c" : ["2", "1"]}}`,
		`{"a":{"b" : ["3", "4" ],"c" : ["2", "1"]}}`,
		`{"a":{"b" : ["3", "4", "5", "6"],"c" : ["2", "1"]}}`)
}

func TestDiffAndPatchError(t *testing.T) {
	checkDiffAndPatchError(t,
		`{"a":1}`,
		`{"a":2}`,
		`{"a":3}`)
	checkDiffAndPatchError(t,
		`{"a":1}`,
		`{"a":2}`,
		`{}`)
	checkDiffAndPatchError(t,
		`1`,
		`2`,
		``)
	checkDiffAndPatchError(t,
		`1`,
		``,
		`2`)
}

type format string

const (
	formatJd    format = "jd"
	formatPatch format = "patch"
	formatMerge format = "merge"
)

func checkDiffAndPatchSuccessSet(t *testing.T, a, b, c, expect string) {
	err := checkDiffAndPatch(t, formatJd, a, b, c, expect, SET)
	if err != nil {
		t.Errorf("Error round-tripping jd format: %v", err)
	}
	// JSON Patch format does not support sets.
}

func checkDiffAndPatchSuccess(t *testing.T, a, b, c, expect string) {
	err := checkDiffAndPatch(t, formatJd, a, b, c, expect)
	if err != nil {
		t.Errorf("Error round-tripping jd format: %v", err)
	}
	err = checkDiffAndPatch(t, formatPatch, a, b, c, expect)
	if err != nil {
		t.Errorf("Error round-tripping patch format: %v", err)
	}
}

func checkDiffAndPatchError(t *testing.T, a, b, c string) {
	err := checkDiffAndPatch(t, formatJd, a, b, c, "")
	if err == nil {
		t.Errorf("Expected error round-tripping jd format.")
	}
	err = checkDiffAndPatch(t, formatPatch, a, b, c, "")
	if err == nil {
		t.Errorf("Expected error rount-tripping patch format.")
	}
}

func checkDiffAndPatch(t *testing.T, f format, a, b, c, expect string, metadata ...Metadata) error {
	nodeA, err := ReadJsonString(a)
	if err != nil {
		return err
	}
	nodeB, err := ReadJsonString(b)
	if err != nil {
		return err
	}
	nodeC, err := ReadJsonString(c)
	if err != nil {
		return err
	}
	expectNode, err := ReadJsonString(expect)
	if err != nil {
		return err
	}
	var diff Diff
	switch f {
	case formatJd:
		diffString := nodeA.Diff(nodeB).Render()
		diff, err = ReadDiffString(diffString)
	case formatPatch:
		patchString, err := nodeA.Diff(nodeB).RenderPatch()
		if err != nil {
			return nil
		}
		diff, err = ReadPatchString(patchString)
	case formatMerge:
		// not yet implemented
	}
	if err != nil {
		return err
	}
	actualNode, err := nodeC.Patch(diff)
	if err != nil {
		return err
	}
	if !actualNode.Equals(expectNode) {
		t.Errorf("actual = %v. Want %v.", actualNode, expectNode)
	}
	return nil
}
