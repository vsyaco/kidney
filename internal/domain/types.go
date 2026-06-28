package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrNoDevice             = errors.New("kindle device is not connected")
	ErrDeviceLocked         = errors.New("kindle device is locked or unavailable")
	ErrDependencyMissing    = errors.New("mtp dependency is missing")
	ErrUnsupportedFileType  = errors.New("unsupported file type")
	ErrUnsupportedFilePath  = errors.New("unsupported file path")
	ErrFileAlreadyExists    = errors.New("file already exists")
	ErrFileNotFound         = errors.New("file not found")
	ErrInvalidFileName      = errors.New("invalid file name")
	ErrNoDocumentsDirectory = errors.New("kindle documents directory was not found")
)

type Device struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Model         string `json:"model,omitempty"`
	Serial        string `json:"serial,omitempty"`
	Backend       string `json:"backend"`
	MountPath     string `json:"mountPath,omitempty"`
	DocumentsPath string `json:"documentsPath,omitempty"`
	Connected     bool   `json:"connected"`
	Message       string `json:"message,omitempty"`
}

type BookFile struct {
	Name      string    `json:"name"`
	Path      string    `json:"path,omitempty"`
	SizeBytes int64     `json:"sizeBytes"`
	Modified  time.Time `json:"modified"`
}

type Transport interface {
	Name() string
	Detect(ctx context.Context) ([]Device, error)
	ListFiles(ctx context.Context, device Device) ([]BookFile, error)
	DownloadFile(ctx context.Context, device Device, fileName string, writer io.Writer) (BookFile, error)
	UploadFile(ctx context.Context, device Device, reader io.Reader, fileName string) (BookFile, error)
	DeleteFile(ctx context.Context, device Device, fileName string) error
	RenameFile(ctx context.Context, device Device, oldName string, newName string) (BookFile, error)
	Unmount(ctx context.Context, device Device) error
}
