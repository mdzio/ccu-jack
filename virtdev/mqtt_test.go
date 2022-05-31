package virtdev

import (
	"strings"
	"testing"
)

func TestExtractorTmpl(t *testing.T) {
	type SubCase struct {
		in  string
		out float64
		err string
	}
	testCases := []struct {
		tmpl     string
		subCases []SubCase
	}{
		{
			`{{with parseJSON .}}{{if .ison}}{{.brightness}}{{else}}0{{end}}{{end}}`,
			[]SubCase{
				{`{"brightness":21,"ison":false}`, 0.0, ""},
				{`{"brightness":42,"ison":true}`, 42.0, ""},
				{`{"brightness":"a","ison":true}`, 0.0, "invalid number literal: a"},
				{`{"ison":true}`, 0.0, "invalid number literal: <no value>"},
				{``, 0.0, "unexpected end of JSON input"},
			},
		},
		{
			`{{if contains . "b"}}1{{else}}0{{end}}`,
			[]SubCase{
				{`abc`, 1.0, ""},
				{`def`, 0.0, ""},
			},
		},
		{
			`{{index (fields .) 1 }}`,
			[]SubCase{
				{`1 2 3`, 2.0, ""},
				{`1`, 0.0, "slice index out of range"},
			},
		},
		{
			`{{index (split . ",") 2 }}`,
			[]SubCase{
				{`1,2,3`, 3.0, ""},
				{`1,2`, 0.0, "slice index out of range"},
			},
		},
		{
			`{{if eq (toLower .) "abc"}}1{{else}}0{{end}}`,
			[]SubCase{
				{`aBC`, 1.0, ""},
				{`def`, 0.0, ""},
			},
		},
		{
			`{{if eq (toUpper .) "ABC"}}1{{else}}0{{end}}`,
			[]SubCase{
				{`Abc`, 1.0, ""},
				{`def`, 0.0, ""},
			},
		},
		{
			`{{if eq (trimSpace .) "abc"}}1{{else}}0{{end}}`,
			[]SubCase{
				{"   abc\t", 1.0, ""},
			},
		},
		{
			`{{slice . 0 2}}{{slice . 3 5}}`,
			[]SubCase{
				{"14:23", 1423.0, ""},
				{"00:01", 1.0, ""},
				{"", 0.0, "index out of range"},
			},
		},
	}
	for _, testCase := range testCases {
		extr, err := newExtractorTmpl(testCase.tmpl)
		if err != nil {
			t.Error(err)
			continue
		}
		for _, subCase := range testCase.subCases {
			v, err := extr.Extract([]byte(subCase.in))
			if subCase.err == "" {
				if err != nil {
					t.Error(err)
					continue
				}
			} else {
				if err == nil {
					t.Errorf("expected error for template %q and input %q", testCase.tmpl, subCase.in)
					continue
				}
				if !strings.Contains(err.Error(), subCase.err) {
					t.Errorf("error %v must contain %q for input %q and template %q", err, subCase.err, subCase.in, testCase.tmpl)
					continue
				}
				continue
			}
			if v != subCase.out {
				t.Errorf("expected value %g for input %q and template %q", subCase.out, subCase.in, testCase.tmpl)
			}
		}
	}
}
