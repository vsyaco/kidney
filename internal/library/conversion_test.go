package library

import "testing"

func TestReplaceExtension(t *testing.T) {
	tests := map[string]string{
		"book.epub":                 "book.azw3",
		"Downloads/Items/book.epub": "Downloads/Items/book.azw3",
		"book.EPUB":                 "book.azw3",
	}

	for input, expected := range tests {
		if got := replaceExtension(input, ".azw3"); got != expected {
			t.Fatalf("replaceExtension(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestBokoPathUsesEnvironmentOverride(t *testing.T) {
	t.Setenv("KIDNEY_BOKO", "/tmp/custom-boko")

	path, err := bokoPath()
	if err != nil {
		t.Fatalf("bokoPath failed: %v", err)
	}

	if path != "/tmp/custom-boko" {
		t.Fatalf("unexpected converter path: %q", path)
	}
}
