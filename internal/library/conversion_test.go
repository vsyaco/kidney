package library

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceExtension(t *testing.T) {
	tests := map[string]string{
		"book.epub":                 "book.azw3",
		"Downloads/Items/book.epub": "Downloads/Items/book.azw3",
		"book.EPUB":                 "book.azw3",
		"book.epub.backup":          "book.epub.azw3",
	}

	for input, expected := range tests {
		if got := replaceExtension(input, ".azw3"); got != expected {
			t.Fatalf("replaceExtension(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestEbookConvertPathUsesPackagedCalibreRuntimeBeforeEnvironmentOverride(t *testing.T) {
	tempDir := t.TempDir()
	toolPath := filepath.Join(tempDir, "tools", "calibre.app", "Contents", "MacOS", "ebook-convert")
	if err := os.MkdirAll(filepath.Dir(toolPath), 0o755); err != nil {
		t.Fatalf("create calibre runtime dir failed: %v", err)
	}
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake ebook-convert failed: %v", err)
	}

	t.Setenv("KIDNEY_EBOOK_CONVERT", "/tmp/custom-ebook-convert")
	useExecutablePath(t, filepath.Join(tempDir, "kidney"))

	got, err := ebookConvertPath()
	if err != nil {
		t.Fatalf("ebookConvertPath failed: %v", err)
	}

	if got != toolPath {
		t.Fatalf("ebookConvertPath = %q, want bundled calibre runtime %q", got, toolPath)
	}
}

func TestEbookConvertPathUsesPackagedFlatToolBeforeEnvironmentOverride(t *testing.T) {
	tempDir := t.TempDir()
	toolPath := filepath.Join(tempDir, "tools", "ebook-convert")
	if err := os.MkdirAll(filepath.Dir(toolPath), 0o755); err != nil {
		t.Fatalf("create tools dir failed: %v", err)
	}
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake ebook-convert failed: %v", err)
	}

	t.Setenv("KIDNEY_EBOOK_CONVERT", "/tmp/custom-ebook-convert")
	useExecutablePath(t, filepath.Join(tempDir, "kidney"))

	got, err := ebookConvertPath()
	if err != nil {
		t.Fatalf("ebookConvertPath failed: %v", err)
	}

	if got != toolPath {
		t.Fatalf("commandPath = %q, want bundled %q", got, toolPath)
	}
}

func TestEbookConvertPathUsesEnvironmentOverride(t *testing.T) {
	t.Setenv("KIDNEY_EBOOK_CONVERT", "/tmp/custom-ebook-convert")
	useExecutablePath(t, filepath.Join(t.TempDir(), "kidney"))

	path, err := ebookConvertPath()
	if err != nil {
		t.Fatalf("ebookConvertPath failed: %v", err)
	}

	if path != "/tmp/custom-ebook-convert" {
		t.Fatalf("unexpected converter path: %q", path)
	}
}

func TestEbookConvertPathUsesPATH(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	toolPath := filepath.Join(binDir, "ebook-convert")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir failed: %v", err)
	}
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake ebook-convert failed: %v", err)
	}

	t.Setenv("PATH", binDir)
	useExecutablePath(t, filepath.Join(tempDir, "kidney"))

	got, err := ebookConvertPath()
	if err != nil {
		t.Fatalf("ebookConvertPath failed: %v", err)
	}

	if got != toolPath {
		t.Fatalf("commandPath = %q, want PATH tool %q", got, toolPath)
	}
}

func TestConvertUploadIfNeededUsesCalibreAZW3(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	toolPath := filepath.Join(binDir, "ebook-convert")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir failed: %v", err)
	}
	script := "#!/bin/sh\ncase \"$2\" in *.azw3) printf converted > \"$2\" ;; *) exit 2 ;; esac\n"
	if err := os.WriteFile(toolPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ebook-convert failed: %v", err)
	}

	t.Setenv("PATH", binDir)
	useExecutablePath(t, filepath.Join(tempDir, "kidney"))

	reader, fileName, cleanup, err := convertUploadIfNeeded(
		context.Background(),
		strings.NewReader("epub content"),
		"Books/book.epub",
	)
	if err != nil {
		t.Fatalf("convertUploadIfNeeded failed: %v", err)
	}
	defer cleanup()

	if fileName != "Books/book.azw3" {
		t.Fatalf("converted fileName = %q, want Books/book.azw3", fileName)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read converted content failed: %v", err)
	}
	if string(content) != "converted" {
		t.Fatalf("unexpected converted content: %q", string(content))
	}
}

func TestConvertUploadIfNeededSkipsPassthroughFormats(t *testing.T) {
	previous := findExecutablePath
	t.Cleanup(func() {
		findExecutablePath = previous
	})

	for _, fileName := range []string{
		"Books/book.pdf",
		"Books/book.mobi",
		"Books/book.azw",
		"Books/book.azw3",
		"Books/book.kfx",
		"Books/book.txt",
	} {
		t.Run(fileName, func(t *testing.T) {
			converterWasResolved := false
			findExecutablePath = func(string) (string, error) {
				converterWasResolved = true
				return "", os.ErrNotExist
			}

			reader, uploadName, cleanup, err := convertUploadIfNeeded(
				context.Background(),
				strings.NewReader("content"),
				fileName,
			)
			if err != nil {
				t.Fatalf("convertUploadIfNeeded failed: %v", err)
			}
			defer cleanup()

			if converterWasResolved {
				t.Fatal("passthrough upload resolved ebook-convert")
			}

			if uploadName != fileName {
				t.Fatalf("uploadName = %q, want %q", uploadName, fileName)
			}

			content, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("read passthrough content failed: %v", err)
			}
			if string(content) != "content" {
				t.Fatalf("unexpected passthrough content: %q", string(content))
			}
		})
	}
}

func useExecutablePath(t *testing.T, executablePath string) {
	t.Helper()

	previous := currentExecutablePath
	currentExecutablePath = func() (string, error) {
		return executablePath, nil
	}
	t.Cleanup(func() {
		currentExecutablePath = previous
	})
}
