package rewards

import (
	"strings"
	"testing"
)

func TestRenderOverlayTemplateSanitizesVariables(t *testing.T) {
	message := renderOverlayTemplate("{viewer} just won {prize}!", "<Kevin>", "<script>")
	if strings.Contains(message, "<script>") || strings.Contains(message, "<Kevin>") {
		t.Fatalf("template output was not escaped: %s", message)
	}
	if !strings.Contains(message, "&lt;Kevin&gt;") {
		t.Fatalf("expected escaped viewer name, got %s", message)
	}
}
