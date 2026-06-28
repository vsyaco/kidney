package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hanwen/go-mtpfs/mtp"

	"kidney/internal/domain"
)

const amazonUSBVendorID = 6473

type MTPTransport struct {
	mu sync.Mutex
}

func NewMTPTransport() *MTPTransport {
	return &MTPTransport{}
}

func (transport *MTPTransport) Name() string {
	return "mtp"
}

func (transport *MTPTransport) Detect(ctx context.Context) ([]domain.Device, error) {
	if _, err := exec.LookPath("mtp-detect"); err == nil {
		output, err := runCommand(ctx, 5*time.Second, "mtp-detect")
		if err != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if err == nil && looksLikeKindleMTP(output) {
			model, serial := parseMTPIdentity(output)
			return []domain.Device{newMTPUSBDevice(model, serial)}, nil
		}
	}

	devices, err := detectUSBKindles(ctx)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

func (transport *MTPTransport) ListFiles(ctx context.Context, device domain.Device) ([]domain.BookFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	transport.mu.Lock()
	defer transport.mu.Unlock()

	client, err := openMTPClient(device)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.ListDocuments()
}

func (transport *MTPTransport) DownloadFile(
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

	transport.mu.Lock()
	defer transport.mu.Unlock()

	client, err := openMTPClient(device)
	if err != nil {
		return domain.BookFile{}, err
	}
	defer client.Close()

	return client.DownloadDocument(fileName, writer)
}

func (transport *MTPTransport) UploadFile(
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

	transport.mu.Lock()
	defer transport.mu.Unlock()

	client, err := openMTPClient(device)
	if err != nil {
		return domain.BookFile{}, err
	}
	defer client.Close()

	return client.UploadDocument(reader, fileName)
}

func (transport *MTPTransport) DeleteFile(ctx context.Context, device domain.Device, fileName string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	transport.mu.Lock()
	defer transport.mu.Unlock()

	client, err := openMTPClient(device)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.DeleteDocument(fileName)
}

func (transport *MTPTransport) RenameFile(
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

	transport.mu.Lock()
	defer transport.mu.Unlock()

	client, err := openMTPClient(device)
	if err != nil {
		return domain.BookFile{}, err
	}
	defer client.Close()

	return client.RenameDocument(oldName, newName)
}

func (transport *MTPTransport) Unmount(ctx context.Context, device domain.Device) error {
	return nil
}

func (transport *MTPTransport) UnmountAll(ctx context.Context) error {
	return nil
}

type mtpClient struct {
	device      *mtp.Device
	storageID   uint32
	documentsID uint32
}

type mtpObject struct {
	handle       uint32
	info         mtp.ObjectInfo
	relativePath string
}

func openMTPClient(device domain.Device) (*mtpClient, error) {
	pattern := ""
	if device.Serial != "" {
		pattern = device.Serial
	}

	mtpDevice, err := mtp.SelectDevice(pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrDeviceLocked, err)
	}

	client := &mtpClient{device: mtpDevice}
	if err := client.device.OpenSession(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("%w: %v", domain.ErrDeviceLocked, err)
	}

	if err := client.resolveDocumentsFolder(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func (client *mtpClient) Close() error {
	if client.device == nil {
		return nil
	}

	err := client.device.Close()
	client.device.Done()
	client.device = nil

	return err
}

func (client *mtpClient) ListDocuments() ([]domain.BookFile, error) {
	objects, err := client.documentsObjectsRecursive()
	if err != nil {
		return nil, err
	}

	files := make([]domain.BookFile, 0, len(objects))
	for _, object := range objects {
		if object.info.ObjectFormat == mtp.OFC_Association || !IsSupportedBookFile(object.info.Filename) {
			continue
		}

		files = append(files, bookFileFromMTPWithPath(object.info, object.relativePath))
	}

	slices.SortFunc(files, func(left domain.BookFile, right domain.BookFile) int {
		return strings.Compare(strings.ToLower(left.Path), strings.ToLower(right.Path))
	})

	return files, nil
}

func (client *mtpClient) UploadDocument(reader io.Reader, fileName string) (domain.BookFile, error) {
	name, err := cleanBookFileName(fileName)
	if err != nil {
		return domain.BookFile{}, err
	}

	if _, err := client.findDocument(name); err == nil {
		return domain.BookFile{}, domain.ErrFileAlreadyExists
	} else if !errors.Is(err, domain.ErrFileNotFound) {
		return domain.BookFile{}, err
	}

	payload, err := io.ReadAll(reader)
	if err != nil {
		return domain.BookFile{}, err
	}

	now := time.Now()
	info := &mtp.ObjectInfo{
		StorageID:        client.storageID,
		ObjectFormat:     objectFormatForFile(name),
		CompressedSize:   uint32(len(payload)),
		ParentObject:     client.documentsID,
		Filename:         name,
		ModificationDate: now,
		CaptureDate:      now,
	}

	_, _, handle, err := client.device.SendObjectInfo(client.storageID, client.documentsID, info)
	if err != nil {
		return domain.BookFile{}, err
	}

	if err := client.device.SendObject(bytes.NewReader(payload), int64(len(payload))); err != nil {
		_ = client.device.DeleteObject(handle)
		return domain.BookFile{}, err
	}

	if err := client.device.GetObjectInfo(handle, info); err != nil {
		return domain.BookFile{}, err
	}

	return bookFileFromMTP(*info), nil
}

func (client *mtpClient) DownloadDocument(fileName string, writer io.Writer) (domain.BookFile, error) {
	object, err := client.findDocument(fileName)
	if err != nil {
		return domain.BookFile{}, err
	}

	if object.info.ObjectFormat == mtp.OFC_Association {
		return domain.BookFile{}, domain.ErrUnsupportedFilePath
	}

	if err := client.device.GetObject(object.handle, writer); err != nil {
		return domain.BookFile{}, err
	}

	return bookFileFromMTPWithPath(object.info, object.relativePath), nil
}

func (client *mtpClient) DeleteDocument(fileName string) error {
	object, err := client.findDocument(fileName)
	if err != nil {
		return err
	}

	if object.info.ObjectFormat == mtp.OFC_Association {
		return domain.ErrUnsupportedFilePath
	}

	return client.device.DeleteObject(object.handle)
}

func (client *mtpClient) RenameDocument(oldName string, newName string) (domain.BookFile, error) {
	_, err := cleanBookFileName(newName)
	if err != nil {
		return domain.BookFile{}, err
	}

	object, err := client.findDocument(oldName)
	if err != nil {
		return domain.BookFile{}, err
	}

	if object.info.ObjectFormat == mtp.OFC_Association {
		return domain.BookFile{}, domain.ErrUnsupportedFilePath
	}

	newPath := joinMTPPath(parentPath(object.relativePath), newName)
	if object.relativePath != newPath {
		if _, err := client.findDocument(newPath); err == nil {
			return domain.BookFile{}, domain.ErrFileAlreadyExists
		} else if !errors.Is(err, domain.ErrFileNotFound) {
			return domain.BookFile{}, err
		}
	}

	if err := client.device.SetObjectPropValue(object.handle, mtp.OPC_ObjectFileName, &mtp.StringValue{Value: newName}); err != nil {
		return domain.BookFile{}, err
	}

	var info mtp.ObjectInfo
	if err := client.device.GetObjectInfo(object.handle, &info); err != nil {
		return domain.BookFile{}, err
	}

	if info.Filename == "" {
		info.Filename = newName
	}

	return bookFileFromMTPWithPath(info, joinMTPPath(parentPath(object.relativePath), info.Filename)), nil
}

func (client *mtpClient) resolveDocumentsFolder() error {
	var storageIDs mtp.Uint32Array
	if err := client.device.GetStorageIDs(&storageIDs); err != nil {
		return err
	}

	for _, storageID := range storageIDs.Values {
		for _, rootParent := range []uint32{0xffffffff, 0} {
			handle, err := client.findFolderInStorage(storageID, rootParent, []string{"documents", "Documents"})
			if err == nil {
				client.storageID = storageID
				client.documentsID = handle
				return nil
			}
		}
	}

	return domain.ErrNoDocumentsDirectory
}

func (client *mtpClient) findFolderInStorage(storageID uint32, parent uint32, names []string) (uint32, error) {
	var handles mtp.Uint32Array
	if err := client.device.GetObjectHandles(storageID, 0, parent, &handles); err != nil {
		return 0, err
	}

	for _, handle := range handles.Values {
		var info mtp.ObjectInfo
		if err := client.device.GetObjectInfo(handle, &info); err != nil {
			continue
		}

		if info.ObjectFormat != mtp.OFC_Association {
			continue
		}

		for _, name := range names {
			if info.Filename == name {
				return handle, nil
			}
		}
	}

	for _, handle := range handles.Values {
		var info mtp.ObjectInfo
		if err := client.device.GetObjectInfo(handle, &info); err != nil || info.ObjectFormat != mtp.OFC_Association {
			continue
		}

		found, err := client.findFolderInStorage(storageID, handle, names)
		if err == nil {
			return found, nil
		}
	}

	return 0, domain.ErrNoDocumentsDirectory
}

func (client *mtpClient) documentsObjects() ([]mtpObject, error) {
	return client.childObjects(client.documentsID, "")
}

func (client *mtpClient) documentsObjectsRecursive() ([]mtpObject, error) {
	return client.childObjectsRecursive(client.documentsID, "")
}

func (client *mtpClient) childObjects(parent uint32, parentPath string) ([]mtpObject, error) {
	var handles mtp.Uint32Array
	if err := client.device.GetObjectHandles(client.storageID, 0, parent, &handles); err != nil {
		return nil, err
	}

	objects := make([]mtpObject, 0, len(handles.Values))
	for _, handle := range handles.Values {
		var info mtp.ObjectInfo
		if err := client.device.GetObjectInfo(handle, &info); err != nil {
			continue
		}

		objects = append(objects, mtpObject{
			handle:       handle,
			info:         info,
			relativePath: joinMTPPath(parentPath, info.Filename),
		})
	}

	return objects, nil
}

func (client *mtpClient) childObjectsRecursive(parent uint32, parentPath string) ([]mtpObject, error) {
	objects, err := client.childObjects(parent, parentPath)
	if err != nil {
		return nil, err
	}

	allObjects := make([]mtpObject, 0, len(objects))
	for _, object := range objects {
		allObjects = append(allObjects, object)
		if object.info.ObjectFormat != mtp.OFC_Association {
			continue
		}

		children, err := client.childObjectsRecursive(object.handle, object.relativePath)
		if err != nil {
			continue
		}

		allObjects = append(allObjects, children...)
	}

	return allObjects, nil
}

func (client *mtpClient) findDocument(fileName string) (mtpObject, error) {
	path, err := cleanMTPRelativePath(fileName)
	if err != nil {
		return mtpObject{}, err
	}

	objects, err := client.documentsObjectsRecursive()
	if err != nil {
		return mtpObject{}, err
	}

	for _, object := range objects {
		if object.relativePath == path {
			return object, nil
		}
	}

	return mtpObject{}, domain.ErrFileNotFound
}

func bookFileFromMTP(info mtp.ObjectInfo) domain.BookFile {
	return bookFileFromMTPWithPath(info, info.Filename)
}

func bookFileFromMTPWithPath(info mtp.ObjectInfo, path string) domain.BookFile {
	return domain.BookFile{
		Name:      info.Filename,
		Path:      path,
		SizeBytes: int64(info.CompressedSize),
		Modified:  info.ModificationDate,
	}
}

func cleanMTPRelativePath(path string) (string, error) {
	cleaned := strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if cleaned == "" || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "/") {
		return "", domain.ErrUnsupportedFilePath
	}

	parts := strings.Split(cleaned, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", domain.ErrUnsupportedFilePath
		}
	}

	if !IsSupportedBookFile(parts[len(parts)-1]) {
		return "", domain.ErrUnsupportedFileType
	}

	return strings.Join(parts, "/"), nil
}

func joinMTPPath(parent string, name string) string {
	if parent == "" {
		return name
	}

	return parent + "/" + name
}

func parentPath(path string) string {
	index := strings.LastIndex(path, "/")
	if index == -1 {
		return ""
	}

	return path[:index]
}

func objectFormatForFile(fileName string) uint16 {
	switch strings.ToLower(fileExtension(fileName)) {
	case ".txt":
		return mtp.OFC_Text
	default:
		return mtp.OFC_MTP_UndefinedDocument
	}
}

func fileExtension(fileName string) string {
	index := strings.LastIndex(fileName, ".")
	if index == -1 {
		return ""
	}

	return fileName[index:]
}

func requireCommand(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s", domain.ErrDependencyMissing, name)
	}

	return nil
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(commandCtx, name, args...)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output

	err := command.Run()
	return output.String(), err
}

func detectUSBKindles(ctx context.Context) ([]domain.Device, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectMacOSUSBKindles(ctx)
	case "linux":
		return detectLinuxUSBKindles(ctx)
	default:
		return nil, nil
	}
}

func detectMacOSUSBKindles(ctx context.Context) ([]domain.Device, error) {
	if _, err := exec.LookPath("ioreg"); err != nil {
		return nil, nil
	}

	output, err := runCommand(ctx, 5*time.Second, "ioreg", "-p", "IOUSB", "-l", "-w", "0")
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, nil
	}

	return parseMacOSUSBKindles(output), nil
}

func detectLinuxUSBKindles(ctx context.Context) ([]domain.Device, error) {
	if _, err := exec.LookPath("lsusb"); err != nil {
		return nil, nil
	}

	output, err := runCommand(ctx, 5*time.Second, "lsusb")
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, nil
	}

	devices := make([]domain.Device, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "kindle") && !strings.Contains(lower, "amazon") && !strings.Contains(lower, "1949:") {
			continue
		}

		devices = append(devices, newMTPUSBDevice(line, ""))
	}

	return devices, nil
}

func parseMacOSUSBKindles(output string) []domain.Device {
	devices := make([]domain.Device, 0)
	var current *macOSUSBDevice

	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "+-o ") && strings.Contains(line, "IOUSBHostDevice") {
			if current != nil && current.isKindle() {
				devices = append(devices, current.device())
			}

			current = &macOSUSBDevice{name: macOSUSBDeviceName(line)}
			continue
		}

		if current == nil {
			continue
		}

		switch {
		case strings.Contains(line, "\"USB Product Name\"") || strings.Contains(line, "\"kUSBProductString\""):
			current.product = valueAfterEquals(line)
		case strings.Contains(line, "\"USB Vendor Name\"") || strings.Contains(line, "\"kUSBVendorString\""):
			current.vendor = valueAfterEquals(line)
		case strings.Contains(line, "\"USB Serial Number\"") || strings.Contains(line, "\"kUSBSerialNumberString\""):
			current.serial = valueAfterEquals(line)
		case strings.Contains(line, "\"idVendor\""):
			current.vendorID = intValueAfterEquals(line)
		}
	}

	if current != nil && current.isKindle() {
		devices = append(devices, current.device())
	}

	return devices
}

type macOSUSBDevice struct {
	name     string
	product  string
	vendor   string
	serial   string
	vendorID int
}

func (device macOSUSBDevice) isKindle() bool {
	text := strings.ToLower(strings.Join([]string{device.name, device.product, device.vendor}, " "))
	return device.vendorID == amazonUSBVendorID ||
		strings.Contains(text, "kindle") ||
		strings.Contains(text, "amazon")
}

func (device macOSUSBDevice) device() domain.Device {
	return newMTPUSBDevice(firstNonEmpty(device.product, device.name), device.serial)
}

func newMTPUSBDevice(model string, serial string) domain.Device {
	name := firstNonEmpty(model, "Kindle MTP")
	id := "mtp-usb:" + name
	if serial != "" {
		id = "mtp-usb:" + serial
	}

	return domain.Device{
		ID:        id,
		Name:      name,
		Model:     model,
		Serial:    serial,
		Backend:   "mtp",
		Connected: true,
		Message:   "USB visible; direct MTP file access is built in.",
	}
}

func macOSUSBDeviceName(line string) string {
	_, after, found := strings.Cut(line, "+-o ")
	if !found {
		return ""
	}

	name, _, found := strings.Cut(after, "@")
	if found {
		return strings.TrimSpace(name)
	}

	name, _, _ = strings.Cut(after, "<")
	return strings.TrimSpace(name)
}

func valueAfterEquals(line string) string {
	_, value, found := strings.Cut(line, "=")
	if !found {
		return ""
	}

	return strings.Trim(strings.TrimSpace(value), `"`)
}

func intValueAfterEquals(line string) int {
	value := valueAfterEquals(line)
	var result int
	_, _ = fmt.Sscanf(value, "%d", &result)
	return result
}

func looksLikeKindleMTP(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "kindle") || strings.Contains(lower, "amazon")
}

func parseMTPIdentity(output string) (string, string) {
	var model string
	var serial string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lower := strings.ToLower(line)

		if model == "" && (strings.Contains(lower, "friendly name") || strings.Contains(lower, "model")) {
			model = valueAfterColon(line)
		}

		if serial == "" && strings.Contains(lower, "serial") {
			serial = valueAfterColon(line)
		}
	}

	return model, serial
}

func valueAfterColon(line string) string {
	_, value, found := strings.Cut(line, ":")
	if !found {
		return ""
	}

	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
