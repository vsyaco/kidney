package transport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vsyaco/kidney/internal/domain"
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
	name, err := cleanBookRelativePath(fileName)
	if err != nil {
		return "", err
	}

	if strings.Contains(name, "/") {
		return "", domain.ErrUnsupportedFilePath
	}

	return name, nil
}

func cleanBookRelativePath(fileName string) (string, error) {
	name := strings.TrimSpace(strings.ReplaceAll(fileName, "\\", "/"))
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, "/") {
		return "", domain.ErrInvalidFileName
	}

	if path.Clean(name) != name {
		return "", domain.ErrUnsupportedFilePath
	}

	parts := strings.Split(name, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", domain.ErrUnsupportedFilePath
		}
	}

	if !IsSupportedBookFile(name) {
		return "", fmt.Errorf("%w: %s", domain.ErrUnsupportedFileType, filepath.Ext(name))
	}

	return name, nil
}

func safeBookPath(root string, fileName string) (string, string, error) {
	name, err := cleanBookRelativePath(fileName)
	if err != nil {
		return "", "", err
	}

	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", "", err
	}

	filePath := filepath.Join(cleanRoot, filepath.FromSlash(name))
	rel, err := filepath.Rel(cleanRoot, filePath)
	if err != nil {
		return "", "", err
	}

	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", "", domain.ErrUnsupportedFilePath
	}

	return filePath, name, nil
}

func listRootFiles(root string) ([]domain.BookFile, error) {
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrNoDocumentsDirectory
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, domain.ErrDeviceLocked
		}
		return nil, err
	}

	files := make([]domain.BookFile, 0)
	err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if entry.IsDir() || !IsSupportedBookFile(entry.Name()) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}

		relativePath, err := filepath.Rel(root, filePath)
		if err != nil {
			return nil
		}
		relativePath = filepath.ToSlash(relativePath)

		files = append(files, domain.BookFile{
			Name:      entry.Name(),
			Path:      relativePath,
			SizeBytes: info.Size(),
			Modified:  info.ModTime(),
		})
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrNoDocumentsDirectory
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, domain.ErrDeviceLocked
		}
		return nil, err
	}

	slices.SortFunc(files, func(left domain.BookFile, right domain.BookFile) int {
		return strings.Compare(strings.ToLower(left.Path), strings.ToLower(right.Path))
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
	oldPath, cleanOldName, err := safeBookPath(root, oldName)
	if err != nil {
		return domain.BookFile{}, err
	}

	cleanNewName, err := cleanBookFileName(newName)
	if err != nil {
		return domain.BookFile{}, err
	}

	newRelativePath := joinRelativePath(parentRelativePath(cleanOldName), cleanNewName)
	newPath := filepath.Join(filepath.Dir(oldPath), cleanNewName)

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
			Path:      newRelativePath,
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
		Path:      newRelativePath,
		SizeBytes: info.Size(),
		Modified:  info.ModTime(),
	}, nil
}

func parentRelativePath(relativePath string) string {
	parent := path.Dir(relativePath)
	if parent == "." {
		return ""
	}

	return parent
}

func joinRelativePath(parent string, name string) string {
	if parent == "" {
		return name
	}

	return parent + "/" + name
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
