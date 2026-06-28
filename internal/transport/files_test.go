package transport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kidney/internal/domain"
)

func TestIsSupportedBookFile(t *testing.T) {
	tests := map[string]bool{
		"book.epub": true,
		"book.EPUB": true,
		"book.pdf":  true,
		"book.mobi": true,
		"book.azw":  true,
		"book.azw3": true,
		"book.kfx":  true,
		"book.txt":  true,
		"book.docx": false,
		"book":      false,
	}

	for fileName, expected := range tests {
		if got := IsSupportedBookFile(fileName); got != expected {
			t.Fatalf("IsSupportedBookFile(%q) = %v, want %v", fileName, got, expected)
		}
	}
}

func TestSafeBookPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	for _, fileName := range []string{"../book.epub", "nested/book.epub", `nested\book.epub`, "..", ""} {
		_, _, err := safeBookPath(root, fileName)
		if err == nil {
			t.Fatalf("expected %q to be rejected", fileName)
		}
	}
}

func TestRootOperationsValidateAndPreventCollisions(t *testing.T) {
	root := t.TempDir()

	book, err := uploadRootFile(root, strings.NewReader("content"), "book.epub")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if book.Name != "book.epub" || book.SizeBytes != int64(len("content")) {
		t.Fatalf("unexpected upload result: %#v", book)
	}

	if _, err := uploadRootFile(root, strings.NewReader("other"), "book.epub"); !errors.Is(err, domain.ErrFileAlreadyExists) {
		t.Fatalf("expected collision, got %v", err)
	}

	if _, err := uploadRootFile(root, strings.NewReader("other"), "book.docx"); !errors.Is(err, domain.ErrUnsupportedFileType) {
		t.Fatalf("expected unsupported type, got %v", err)
	}

	if _, err := renameRootFile(root, "book.epub", "renamed.pdf"); err != nil {
		t.Fatalf("rename failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "renamed.pdf")); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}

	if err := deleteRootFile(root, "renamed.pdf"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if err := deleteRootFile(root, "renamed.pdf"); !errors.Is(err, domain.ErrFileNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDeleteRejectsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "folder.epub"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	if err := deleteRootFile(root, "folder.epub"); !errors.Is(err, domain.ErrUnsupportedFilePath) {
		t.Fatalf("expected unsupported path, got %v", err)
	}
}

func TestDiskTransportDetectsKindleLikeVolume(t *testing.T) {
	scanRoot := t.TempDir()
	mountRoot := filepath.Join(scanRoot, "Kindle")
	if err := os.MkdirAll(filepath.Join(mountRoot, "documents"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	transport := NewDiskTransportWithRoots([]string{scanRoot})
	devices, err := transport.Detect(context.Background())
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected one device, got %#v", devices)
	}

	if devices[0].DocumentsPath != filepath.Join(mountRoot, "documents") {
		t.Fatalf("unexpected documents path: %#v", devices[0])
	}
}

func TestListRootFilesFiltersUnsupportedAndDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "book.epub"), []byte("book"), 0o644); err != nil {
		t.Fatalf("write epub failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.docx"), []byte("doc"), 0o644); err != nil {
		t.Fatalf("write docx failed: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "folder.pdf"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	files, err := listRootFiles(root)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(files) != 1 || files[0].Name != "book.epub" {
		t.Fatalf("unexpected files: %#v", files)
	}
}
