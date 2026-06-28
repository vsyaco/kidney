package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"runtime"
	"strings"

	"github.com/vsyaco/kidney/internal/domain"
)

type Service struct {
	transports []domain.Transport
}

type ConnectedDevice struct {
	Device    domain.Device
	Transport domain.Transport
}

type Status struct {
	Connected bool            `json:"connected"`
	Device    *domain.Device  `json:"device,omitempty"`
	Devices   []domain.Device `json:"devices"`
	Error     string          `json:"error,omitempty"`
}

func NewService(transports []domain.Transport) *Service {
	return &Service{transports: transports}
}

func (service *Service) Status(ctx context.Context) Status {
	devices, err := service.Devices(ctx)
	if err != nil {
		return Status{
			Connected: false,
			Devices:   devices,
			Error:     FriendlyError(err),
		}
	}

	if len(devices) == 0 {
		return Status{
			Connected: false,
			Devices:   devices,
			Error:     FriendlyError(domain.ErrNoDevice),
		}
	}

	return Status{
		Connected: true,
		Device:    &devices[0],
		Devices:   devices,
	}
}

func (service *Service) Devices(ctx context.Context) ([]domain.Device, error) {
	allDevices := make([]domain.Device, 0)
	var firstErr error

	for _, transport := range service.transports {
		devices, err := transport.Detect(ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		allDevices = append(allDevices, devices...)
	}

	if len(allDevices) > 0 {
		return allDevices, nil
	}

	return allDevices, firstErr
}

func (service *Service) List(ctx context.Context) ([]domain.BookFile, error) {
	device, err := service.connectedDevice(ctx)
	if err != nil {
		return nil, err
	}

	return device.Transport.ListFiles(ctx, device.Device)
}

func (service *Service) Upload(ctx context.Context, reader io.Reader, fileName string) (domain.BookFile, error) {
	device, err := service.connectedDevice(ctx)
	if err != nil {
		return domain.BookFile{}, err
	}

	uploadName, err := service.defaultUploadName(ctx, device, fileName)
	if err != nil {
		return domain.BookFile{}, err
	}

	uploadReader, uploadName, cleanup, err := convertUploadIfNeeded(ctx, reader, uploadName)
	if err != nil {
		return domain.BookFile{}, err
	}
	defer cleanup()

	return device.Transport.UploadFile(ctx, device.Device, uploadReader, uploadName)
}

func (service *Service) Download(ctx context.Context, fileName string, writer io.Writer) (domain.BookFile, error) {
	device, err := service.connectedDevice(ctx)
	if err != nil {
		return domain.BookFile{}, err
	}

	return device.Transport.DownloadFile(ctx, device.Device, fileName, writer)
}

func (service *Service) Delete(ctx context.Context, fileName string) error {
	device, err := service.connectedDevice(ctx)
	if err != nil {
		return err
	}

	return device.Transport.DeleteFile(ctx, device.Device, fileName)
}

func (service *Service) Rename(ctx context.Context, oldName string, newName string) (domain.BookFile, error) {
	device, err := service.connectedDevice(ctx)
	if err != nil {
		return domain.BookFile{}, err
	}

	return device.Transport.RenameFile(ctx, device.Device, oldName, newName)
}

func (service *Service) Unmount(ctx context.Context) error {
	device, err := service.connectedDevice(ctx)
	if err != nil {
		return err
	}

	return device.Transport.Unmount(ctx, device.Device)
}

func (service *Service) connectedDevice(ctx context.Context) (ConnectedDevice, error) {
	var firstErr error

	for _, transport := range service.transports {
		devices, err := transport.Detect(ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if len(devices) > 0 {
			return ConnectedDevice{
				Device:    devices[0],
				Transport: transport,
			}, nil
		}
	}

	if firstErr != nil {
		return ConnectedDevice{}, firstErr
	}

	return ConnectedDevice{}, domain.ErrNoDevice
}

func (service *Service) defaultUploadName(ctx context.Context, device ConnectedDevice, fileName string) (string, error) {
	if hasRelativeFolder(fileName) {
		return fileName, nil
	}

	files, err := device.Transport.ListFiles(ctx, device.Device)
	if err != nil {
		return "", err
	}

	folder := dominantBookFolder(files)
	if folder == "" {
		return fileName, nil
	}

	return path.Join(folder, fileName), nil
}

func dominantBookFolder(files []domain.BookFile) string {
	counts := make(map[string]int)
	bestFolder := ""
	bestCount := 0

	for _, file := range files {
		filePath := firstNonEmpty(file.Path, file.Name)
		folder := parentFolder(filePath)
		counts[folder]++

		count := counts[folder]
		if count > bestCount || count == bestCount && folder != "" && (bestFolder == "" || folder < bestFolder) {
			bestFolder = folder
			bestCount = count
		}
	}

	return bestFolder
}

func hasRelativeFolder(fileName string) bool {
	return strings.ContainsAny(strings.TrimSpace(fileName), `/\`)
}

func parentFolder(filePath string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(filePath, "\\", "/"))
	if normalized == "" {
		return ""
	}

	folder := path.Dir(normalized)
	if folder == "." {
		return ""
	}

	return folder
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func FriendlyError(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, domain.ErrNoDevice):
		return "Kindle is not connected."
	case errors.Is(err, domain.ErrDeviceLocked):
		return "Kindle is locked, disconnected, or unavailable. Unlock it and reconnect USB."
	case errors.Is(err, domain.ErrDependencyMissing):
		return dependencyErrorMessage(err)
	case errors.Is(err, domain.ErrUnsupportedFilePath):
		return "Unsupported file path. Use a plain file name inside the Kindle documents folder."
	case errors.Is(err, domain.ErrUnsupportedFileType):
		return "Unsupported file type. Use EPUB, PDF, MOBI, AZW, AZW3, KFX, or TXT."
	case errors.Is(err, domain.ErrFileAlreadyExists):
		return "A file with that name already exists on the Kindle."
	case errors.Is(err, domain.ErrFileNotFound):
		return "File was not found on the Kindle."
	case errors.Is(err, domain.ErrInvalidFileName):
		return "Invalid file name."
	case errors.Is(err, domain.ErrNoDocumentsDirectory):
		return "Kindle documents folder was not found."
	default:
		return err.Error()
	}
}

func dependencyErrorMessage(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "ebook-convert"):
		return "EPUB conversion runtime missing: ebook-convert. " + installHint(
			"Install Calibre or set KIDNEY_EBOOK_CONVERT to the ebook-convert binary.",
			"Install Calibre or set KIDNEY_EBOOK_CONVERT to the ebook-convert binary.",
		)
	case strings.Contains(message, "simple-mtpfs"):
		return "MTP filesystem dependency missing: simple-mtpfs. " + installHint(
			"Install with: brew install simple-mtpfs && brew install --cask macfuse.",
			"Install with: sudo apt install simple-mtpfs fuse.",
		)
	case strings.Contains(message, "mtp-detect"):
		return "MTP debug dependency missing: mtp-detect. " + installHint(
			"Install with: brew install libmtp.",
			"Install with: sudo apt install mtp-tools.",
		)
	default:
		return fmt.Sprintf("MTP dependency missing. %v", err)
	}
}

func installHint(darwin string, linux string) string {
	if runtime.GOOS == "darwin" {
		return darwin
	}

	if runtime.GOOS == "linux" {
		return linux
	}

	return "Install the missing command for your operating system."
}
