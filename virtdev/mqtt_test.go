package virtdev

import "testing"

func TestExtractorTmpl(t *testing.T) {
	ex, err := newExtractorTmpl(`{{with parseJSON .}}{{if .ison}}{{.brightness}}{{else}}0{{end}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ex.Extract([]byte(`{"brightness":21,"ison":false}`))
	if err != nil {
		t.Fatal(err)
	}
	if v != 0.0 {
		t.Fatal(v)
	}
	v, err = ex.Extract([]byte(`{"brightness":42,"ison":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if v != 42.0 {
		t.Fatal(v)
	}
}
