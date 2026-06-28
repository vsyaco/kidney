package library

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/vsyaco/kidney/internal/domain"
)

func convertUploadIfNeeded(ctx context.Context, reader io.Reader, fileName string) (io.Reader, string, func(), error) {
	if strings.ToLower(path.Ext(fileName)) != ".epub" {
		return reader, fileName, func() {}, nil
	}

	converterPath, err := bokoPath()
	if err != nil {
		return nil, "", nil, fmt.Errorf("%w: boko", domain.ErrDependencyMissing)
	}

	tempDir, err := os.MkdirTemp("", "kidney-epub-convert-*")
	if err != nil {
		return nil, "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	inputPath := filepath.Join(tempDir, "input.epub")
	outputPath := filepath.Join(tempDir, "output.azw3")

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

	command := exec.CommandContext(ctx, converterPath, "convert", inputPath, outputPath)
	output, err := command.CombinedOutput()
	if err != nil {
		cleanup()
		return nil, "", nil, fmt.Errorf("EPUB conversion failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}

	return outputFile, replaceExtension(fileName, ".azw3"), func() {
		_ = outputFile.Close()
		cleanup()
	}, nil
}

func replaceExtension(fileName string, extension string) string {
	return strings.TrimSuffix(fileName, path.Ext(fileName)) + extension
}

func bokoPath() (string, error) {
	if configuredPath := strings.TrimSpace(os.Getenv("KIDNEY_BOKO")); configuredPath != "" {
		return configuredPath, nil
	}

	if bundledPath, ok := bundledBokoPath(); ok {
		return bundledPath, nil
	}

	return exec.LookPath("boko")
}

func bundledBokoPath() (string, bool) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", false
	}

	converterPath := filepath.Join(
		filepath.Dir(executablePath),
		"tools",
		"boko",
	)

	info, err := os.Stat(converterPath)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", false
	}

	return converterPath, true
}
