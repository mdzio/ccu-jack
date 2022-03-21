package virtdev

import "testing"

func TestExtractorTmpl(t *testing.T) {
	ex, err := newExtractorTmpl(`{{with parseJSON .}}{{if eq .turn "off"}}0{{else}}{{.brightness}}{{end}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ex.Extract([]byte(`{"brightness":21,"turn":"off"}`))
	if err != nil {
		t.Fatal(err)
	}
	if v != 0.0 {
		t.Fatal(v)
	}
	v, err = ex.Extract([]byte(`{"brightness":42,"turn":"on"}`))
	if err != nil {
		t.Fatal(err)
	}
	if v != 42.0 {
		t.Fatal(v)
	}
}
