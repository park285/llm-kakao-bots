package prompt

import "testing"

func TestFormatTemplate(t *testing.T) {
	output, err := FormatTemplate("Hello {name} {{test}}", map[string]string{"name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "Hello Alice {test}" {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestFormatTemplateMissingKey(t *testing.T) {
	if _, err := FormatTemplate("Hello {name}", map[string]string{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFormatTemplateInvalidSyntax(t *testing.T) {
	if _, err := FormatTemplate("Hello {name", map[string]string{"name": "A"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateSystemStatic(t *testing.T) {
	if err := ValidateSystemStatic("sys", "Hello {name}"); err == nil {
		t.Fatalf("expected error")
	}
	if err := ValidateSystemStatic("sys", "Hello {{name}}!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEscapeXML(t *testing.T) {
	input := `<tag>&"'`
	expected := "&lt;tag&gt;&amp;&quot;&apos;"
	if got := EscapeXML(input); got != expected {
		t.Fatalf("unexpected escape result: %s", got)
	}
}

func TestWrapXML(t *testing.T) {
	got := WrapXML("q", `a<b`)
	if got != "<q>a&lt;b</q>" {
		t.Fatalf("unexpected wrap result: %s", got)
	}
}
