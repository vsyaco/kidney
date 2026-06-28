package library

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"kidney/internal/domain"
)

type fakeTransport struct {
	name       string
	devices    []domain.Device
	detectErr  error
	files      []domain.BookFile
	uploaded   string
	deleted    string
	renamedOld string
	renamedNew string
}

func (transport *fakeTransport) Name() string {
	return transport.name
}

func (transport *fakeTransport) Detect(context.Context) ([]domain.Device, error) {
	return transport.devices, transport.detectErr
}

func (transport *fakeTransport) ListFiles(context.Context, domain.Device) ([]domain.BookFile, error) {
	return transport.files, nil
}

func (transport *fakeTransport) UploadFile(
	_ context.Context,
	_ domain.Device,
	reader io.Reader,
	fileName string,
) (domain.BookFile, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return domain.BookFile{}, err
	}

	transport.uploaded = fileName + ":" + string(content)
	return domain.BookFile{Name: fileName, SizeBytes: int64(len(content)), Modified: time.Now()}, nil
}

func (transport *fakeTransport) DownloadFile(
	_ context.Context,
	_ domain.Device,
	fileName string,
	writer io.Writer,
) (domain.BookFile, error) {
	if _, err := writer.Write([]byte("content")); err != nil {
		return domain.BookFile{}, err
	}

	return domain.BookFile{Name: fileName, SizeBytes: int64(len("content")), Modified: time.Now()}, nil
}

func (transport *fakeTransport) DeleteFile(_ context.Context, _ domain.Device, fileName string) error {
	transport.deleted = fileName
	return nil
}

func (transport *fakeTransport) RenameFile(
	_ context.Context,
	_ domain.Device,
	oldName string,
	newName string,
) (domain.BookFile, error) {
	transport.renamedOld = oldName
	transport.renamedNew = newName
	return domain.BookFile{Name: newName}, nil
}

func (transport *fakeTransport) Unmount(context.Context, domain.Device) error {
	return nil
}

func TestServiceUsesFirstConnectedTransport(t *testing.T) {
	first := &fakeTransport{name: "disk"}
	second := &fakeTransport{
		name:    "mtp",
		devices: []domain.Device{{ID: "1", Name: "Kindle", Backend: "mtp", Connected: true}},
		files:   []domain.BookFile{{Name: "book.epub"}},
	}

	service := NewService([]domain.Transport{first, second})

	files, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(files) != 1 || files[0].Name != "book.epub" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func TestServiceReportsNoDevice(t *testing.T) {
	service := NewService([]domain.Transport{&fakeTransport{name: "disk"}})

	_, err := service.List(context.Background())
	if err == nil {
		t.Fatal("expected no device error")
	}

	if got := FriendlyError(err); got != "Kindle is not connected." {
		t.Fatalf("unexpected friendly error: %q", got)
	}
}

func TestServiceDelegatesMutations(t *testing.T) {
	device := domain.Device{ID: "1", Name: "Kindle", Backend: "fake", Connected: true}
	transport := &fakeTransport{name: "fake", devices: []domain.Device{device}}
	service := NewService([]domain.Transport{transport})

	if _, err := service.Upload(context.Background(), strings.NewReader("content"), "book.pdf"); err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if transport.uploaded != "book.pdf:content" {
		t.Fatalf("unexpected upload: %q", transport.uploaded)
	}

	if _, err := service.Rename(context.Background(), "book.pdf", "renamed.pdf"); err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if transport.renamedOld != "book.pdf" || transport.renamedNew != "renamed.pdf" {
		t.Fatalf("unexpected rename: %q -> %q", transport.renamedOld, transport.renamedNew)
	}

	if err := service.Delete(context.Background(), "renamed.pdf"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if transport.deleted != "renamed.pdf" {
		t.Fatalf("unexpected delete: %q", transport.deleted)
	}
}
