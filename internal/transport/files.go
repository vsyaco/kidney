package transport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"kidney/internal/domain"
)

var supportedExtensions = []string{
	".azw",
	".azw3",
	".epub",
	".kfx",
	".mobi",
	".pdf",
	".txt",
}

func SupportedExtensions() []string {
	return slices.Clone(supportedExtensions)
}

func IsSupportedBookFile(fileName string) bool {
	return slices.Contains(supportedExtensions, strings.ToLower(filepath.Ext(fileName)))
}

func cleanBookFileName(fileName string) (string, error) {
	name := strings.TrimSpace(fileName)
	if name == "" || name == "." || name == ".." {
		return "", domain.ErrInvalidFileName
	}

	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return "", domain.ErrUnsupportedFilePath
	}

	if filepath.Clean(name) != name {
		return "", domain.ErrUnsupportedFilePath
	}

	if !IsSupportedBookFile(name) {
		return "", fmt.Errorf("%w: %s", domain.ErrUnsupportedFileType, filepath.Ext(name))
	}

	return name, nil
}

func safeBookPath(root string, fileName string) (string, string, error) {
	name, err := cleanBookFileName(fileName)
	if err != nil {
		return "", "", err
	}

	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", "", err
	}

	path := filepath.Join(cleanRoot, name)
	rel, err := filepath.Rel(cleanRoot, path)
	if err != nil {
		return "", "", err
	}

	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", "", domain.ErrUnsupportedFilePath
	}

	return path, name, nil
}

func listRootFiles(root string) ([]domain.BookFile, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrNoDocumentsDirectory
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, domain.ErrDeviceLocked
		}
		return nil, err
	}

	files := make([]domain.BookFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !IsSupportedBookFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, domain.BookFile{
			Name:      entry.Name(),
			Path:      filepath.Join(root, entry.Name()),
			SizeBytes: info.Size(),
			Modified:  info.ModTime(),
		})
	}

	slices.SortFunc(files, func(left domain.BookFile, right domain.BookFile) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})

	return files, nil
}

func uploadRootFile(root string, reader io.Reader, fileName string) (domain.BookFile, error) {
	path, cleanName, err := safeBookPath(root, fileName)
	if err != nil {
		return domain.BookFile{}, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return domain.BookFile{}, domain.ErrFileAlreadyExists
		}
		if errors.Is(err, os.ErrPermission) {
			return domain.BookFile{}, domain.ErrDeviceLocked
		}
		return domain.BookFile{}, err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		_ = os.Remove(path)
		return domain.BookFile{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return domain.BookFile{}, err
	}

	return domain.BookFile{
		Name:      cleanName,
		Path:      path,
		SizeBytes: info.Size(),
		Modified:  info.ModTime(),
	}, nil
}

func downloadRootFile(root string, fileName string, writer io.Writer) (domain.BookFile, error) {
	path, cleanName, err := safeBookPath(root, fileName)
	if err != nil {
		return domain.BookFile{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.BookFile{}, domain.ErrFileNotFound
		}
		if errors.Is(err, os.ErrPermission) {
			return domain.BookFile{}, domain.ErrDeviceLocked
		}
		return domain.BookFile{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return domain.BookFile{}, err
	}
	if info.IsDir() {
		return domain.BookFile{}, domain.ErrUnsupportedFilePath
	}

	if _, err := io.Copy(writer, file); err != nil {
		return domain.BookFile{}, err
	}

	return domain.BookFile{
		Name:      cleanName,
		Path:      cleanName,
		SizeBytes: info.Size(),
		Modified:  info.ModTime(),
	}, nil
}

func deleteRootFile(root string, fileName string) error {
	path, _, err := safeBookPath(root, fileName)
	if err != nil {
		return err
	}

	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.ErrFileNotFound
		}
		if errors.Is(err, os.ErrPermission) {
			return domain.ErrDeviceLocked
		}
		return err
	}

	if info.IsDir() {
		return domain.ErrUnsupportedFilePath
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return domain.ErrDeviceLocked
		}
		return err
	}

	return nil
}

func renameRootFile(root string, oldName string, newName string) (domain.BookFile, error) {
	oldPath, _, err := safeBookPath(root, oldName)
	if err != nil {
		return domain.BookFile{}, err
	}

	newPath, cleanNewName, err := safeBookPath(root, newName)
	if err != nil {
		return domain.BookFile{}, err
	}

	if oldPath == newPath {
		info, err := os.Stat(oldPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return domain.BookFile{}, domain.ErrFileNotFound
			}
			return domain.BookFile{}, err
		}

		return domain.BookFile{
			Name:      cleanNewName,
			Path:      newPath,
			SizeBytes: info.Size(),
			Modified:  info.ModTime(),
		}, nil
	}

	oldInfo, err := os.Lstat(oldPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.BookFile{}, domain.ErrFileNotFound
		}
		return domain.BookFile{}, err
	}
	if oldInfo.IsDir() {
		return domain.BookFile{}, domain.ErrUnsupportedFilePath
	}

	if _, err := os.Lstat(newPath); err == nil {
		return domain.BookFile{}, domain.ErrFileAlreadyExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return domain.BookFile{}, err
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return domain.BookFile{}, domain.ErrDeviceLocked
		}
		return domain.BookFile{}, err
	}

	info, err := os.Stat(newPath)
	if err != nil {
		return domain.BookFile{}, err
	}

	return domain.BookFile{
		Name:      cleanNewName,
		Path:      newPath,
		SizeBytes: info.Size(),
		Modified:  info.ModTime(),
	}, nil
}

func documentsRoot(root string) (string, error) {
	for _, candidate := range []string{"documents", "Documents", "books", "Books"} {
		path := filepath.Join(root, candidate)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, nil
		}
	}

	return "", domain.ErrNoDocumentsDirectory
}
