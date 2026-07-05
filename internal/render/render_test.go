package render

import "testing"

func TestRenderExecutesTextTemplate(t *testing.T) {
	out, err := Render("hello", "hello {{ .Name }}", map[string]string{"Name": "transform"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if string(out) != "hello transform" {
		t.Fatalf("rendered = %q, want hello transform", string(out))
	}
}
