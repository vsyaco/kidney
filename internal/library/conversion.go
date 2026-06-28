package library

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vsyaco/kidney/internal/domain"
)

const (
	ebookConvertCommandName = "ebook-convert"
	ebookConvertPathEnv     = "KIDNEY_EBOOK_CONVERT"
	epubOutputExtension     = ".azw3"
)

var (
	currentExecutablePath = os.Executable
	findExecutablePath    = exec.LookPath
)

func convertUploadIfNeeded(ctx context.Context, reader io.Reader, fileName string) (io.Reader, string, func(), error) {
	if strings.ToLower(path.Ext(fileName)) != ".epub" {
		return reader, fileName, func() {}, nil
	}

	converterPath, err := ebookConvertPath()
	if err != nil {
		return nil, "", nil, fmt.Errorf("%w: %s", domain.ErrDependencyMissing, ebookConvertCommandName)
	}

	tempDir, err := os.MkdirTemp("", "kidney-epub-convert-*")
	if err != nil {
		return nil, "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	inputPath := filepath.Join(tempDir, "input.epub")
	outputPath := filepath.Join(tempDir, "output"+epubOutputExtension)

	inputFile, err := os.Create(inputPath)
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}

	if _, err := io.Copy(inputFile, reader); err != nil {
		_ = inputFile.Close()
		cleanup()
		return nil, "", nil, err
	}

	if err := inputFile.Close(); err != nil {
		cleanup()
		return nil, "", nil, err
	}

	command := exec.CommandContext(ctx, converterPath, inputPath, outputPath)
	output, err := command.CombinedOutput()
	if err != nil {
		cleanup()
		return nil, "", nil, fmt.Errorf(
			"EPUB conversion failed with %s: %w: %s",
			ebookConvertCommandName,
			err,
			strings.TrimSpace(string(output)),
		)
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}

	return outputFile, replaceExtension(fileName, epubOutputExtension), func() {
		_ = outputFile.Close()
		cleanup()
	}, nil
}

func replaceExtension(fileName string, extension string) string {
	return strings.TrimSuffix(fileName, path.Ext(fileName)) + extension
}

func ebookConvertPath() (string, error) {
	for _, bundledPath := range bundledEbookConvertPaths() {
		if isExecutableFile(bundledPath) {
			return bundledPath, nil
		}
	}

	if bundledPath, ok := bundledToolPath(ebookConvertCommandName); ok {
		return bundledPath, nil
	}

	if configuredPath := strings.TrimSpace(os.Getenv(ebookConvertPathEnv)); configuredPath != "" {
		return configuredPath, nil
	}

	return findExecutablePath(ebookConvertCommandName)
}

func bundledEbookConvertPaths() []string {
	executablePath, err := currentExecutablePath()
	if err != nil {
		return nil
	}

	executableDir := filepath.Dir(executablePath)

	paths := []string{
		filepath.Join(
			executableDir,
			"tools",
			"calibre.app",
			"Contents",
			"MacOS",
			ebookConvertCommandName,
		),
		filepath.Join(executableDir, "tools", "calibre", ebookConvertCommandName),
	}

	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(executableDir, "tools", "calibre", ebookConvertCommandName+".exe"))
	}

	return paths
}

func bundledToolPath(commandName string) (string, bool) {
	executablePath, err := currentExecutablePath()
	if err != nil {
		return "", false
	}

	toolPaths := []string{filepath.Join(filepath.Dir(executablePath), "tools", commandName)}
	if runtime.GOOS == "windows" {
		toolPaths = append(toolPaths, filepath.Join(filepath.Dir(executablePath), "tools", commandName+".exe"))
	}

	for _, toolPath := range toolPaths {
		if isExecutableFile(toolPath) {
			return toolPath, true
		}
	}

	return "", false
}

func isExecutableFile(filePath string) bool {
	info, err := os.Stat(filePath)

	if err != nil || info.IsDir() {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	return info.Mode()&0o111 != 0
}
