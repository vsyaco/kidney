package transport

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"kidney/internal/domain"
)

type DiskTransport struct {
	roots []string
}

func NewDiskTransport() *DiskTransport {
	return &DiskTransport{roots: diskScanRoots()}
}

func NewDiskTransportWithRoots(roots []string) *DiskTransport {
	return &DiskTransport{roots: roots}
}

func (transport *DiskTransport) Name() string {
	return "disk"
}

func (transport *DiskTransport) Detect(ctx context.Context) ([]domain.Device, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	devices := make([]domain.Device, 0)
	for _, scanRoot := range transport.roots {
		entries, err := os.ReadDir(scanRoot)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			mountPath := filepath.Join(scanRoot, entry.Name())
			docRoot, err := documentsRoot(mountPath)
			if err != nil {
				continue
			}

			if !looksLikeKindleVolume(entry.Name(), mountPath) {
				continue
			}

			devices = append(devices, domain.Device{
				ID:            "disk:" + mountPath,
				Name:          entry.Name(),
				Model:         entry.Name(),
				Backend:       transport.Name(),
				MountPath:     mountPath,
				DocumentsPath: docRoot,
				Connected:     true,
			})
		}
	}

	return devices, nil
}

func (transport *DiskTransport) ListFiles(ctx context.Context, device domain.Device) ([]domain.BookFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return listRootFiles(device.DocumentsPath)
}

func (transport *DiskTransport) UploadFile(
	ctx context.Context,
	device domain.Device,
	reader io.Reader,
	fileName string,
) (domain.BookFile, error) {
	select {
	case <-ctx.Done():
		return domain.BookFile{}, ctx.Err()
	default:
	}

	return uploadRootFile(device.DocumentsPath, reader, fileName)
}

func (transport *DiskTransport) DownloadFile(
	ctx context.Context,
	device domain.Device,
	fileName string,
	writer io.Writer,
) (domain.BookFile, error) {
	select {
	case <-ctx.Done():
		return domain.BookFile{}, ctx.Err()
	default:
	}

	return downloadRootFile(device.DocumentsPath, fileName, writer)
}

func (transport *DiskTransport) DeleteFile(ctx context.Context, device domain.Device, fileName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return deleteRootFile(device.DocumentsPath, fileName)
}

func (transport *DiskTransport) RenameFile(
	ctx context.Context,
	device domain.Device,
	oldName string,
	newName string,
) (domain.BookFile, error) {
	select {
	case <-ctx.Done():
		return domain.BookFile{}, ctx.Err()
	default:
	}

	return renameRootFile(device.DocumentsPath, oldName, newName)
}

func (transport *DiskTransport) Unmount(ctx context.Context, device domain.Device) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}

func diskScanRoots() []string {
	if runtime.GOOS == "darwin" {
		return []string{"/Volumes"}
	}

	if runtime.GOOS == "linux" {
		user := os.Getenv("USER")
		roots := []string{"/mnt", "/media"}
		if user != "" {
			roots = append(roots, filepath.Join("/media", user), filepath.Join("/run/media", user))
		}
		return roots
	}

	return []string{}
}

func looksLikeKindleVolume(name string, mountPath string) bool {
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "kindle") || strings.Contains(lowerName, "amazon") {
		return true
	}

	for _, marker := range []string{"audible", "system"} {
		info, err := os.Stat(filepath.Join(mountPath, marker))
		if err == nil && info.IsDir() {
			return true
		}
	}

	return false
}
