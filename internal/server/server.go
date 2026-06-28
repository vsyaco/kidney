package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"kidney/internal/domain"
	"kidney/internal/library"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	service *library.Service
	mux     *http.ServeMux
}

type renameRequest struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(service *library.Service) (*Server, error) {
	server := &Server{
		service: service,
		mux:     http.NewServeMux(),
	}

	if err := server.routes(); err != nil {
		return nil, err
	}

	return server, nil
}

func (server *Server) Handler() http.Handler {
	return server.mux
}

func (server *Server) routes() error {
	staticRoot, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}

	server.mux.Handle("GET /", http.FileServer(http.FS(staticRoot)))
	server.mux.HandleFunc("GET /api/status", server.handleStatus)
	server.mux.HandleFunc("GET /api/books", server.handleBooks)
	server.mux.HandleFunc("GET /api/download", server.handleDownload)
	server.mux.HandleFunc("POST /api/upload", server.handleUpload)
	server.mux.HandleFunc("PATCH /api/books", server.handleRename)
	server.mux.HandleFunc("DELETE /api/books", server.handleDelete)
	server.mux.HandleFunc("POST /api/unmount", server.handleUnmount)

	return nil
}

func (server *Server) handleStatus(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, server.service.Status(request.Context()))
}

func (server *Server) handleBooks(writer http.ResponseWriter, request *http.Request) {
	files, err := server.service.List(request.Context())
	if err != nil {
		writeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, files)
}

func (server *Server) handleUpload(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseMultipartForm(128 << 20); err != nil {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "Upload request is invalid."})
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "Choose a local file to upload."})
		return
	}
	defer file.Close()

	fileName := strings.TrimSpace(request.FormValue("name"))
	if fileName == "" {
		fileName = header.Filename
	}

	book, err := server.service.Upload(request.Context(), file, fileName)
	if err != nil {
		writeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, book)
}

func (server *Server) handleDownload(writer http.ResponseWriter, request *http.Request) {
	fileName := request.URL.Query().Get("name")
	if fileName == "" {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "File name is required."})
		return
	}

	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Content-Disposition", contentDisposition(fileName))

	if _, err := server.service.Download(request.Context(), fileName, writer); err != nil {
		writeError(writer, err)
		return
	}
}

func (server *Server) handleRename(writer http.ResponseWriter, request *http.Request) {
	var payload renameRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "Rename request is invalid."})
		return
	}

	book, err := server.service.Rename(request.Context(), payload.OldName, payload.NewName)
	if err != nil {
		writeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, book)
}

func (server *Server) handleDelete(writer http.ResponseWriter, request *http.Request) {
	fileName := request.URL.Query().Get("name")
	if fileName == "" {
		writeJSON(writer, http.StatusBadRequest, errorResponse{Error: "File name is required."})
		return
	}

	if err := server.service.Delete(request.Context(), fileName); err != nil {
		writeError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (server *Server) handleUnmount(writer http.ResponseWriter, request *http.Request) {
	if err := server.service.Unmount(request.Context()); err != nil {
		writeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{"status": "unmounted"})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}

	if err := json.NewEncoder(writer).Encode(value); err != nil {
		http.Error(writer, fmt.Sprintf("response encode failed: %v", err), http.StatusInternalServerError)
	}
}

func writeError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNoDevice), errors.Is(err, domain.ErrDeviceLocked):
		status = http.StatusServiceUnavailable
	case errors.Is(err, domain.ErrDependencyMissing):
		status = http.StatusFailedDependency
	case errors.Is(err, domain.ErrUnsupportedFilePath),
		errors.Is(err, domain.ErrUnsupportedFileType),
		errors.Is(err, domain.ErrInvalidFileName):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrFileAlreadyExists):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrFileNotFound):
		status = http.StatusNotFound
	}

	writeJSON(writer, status, errorResponse{Error: library.FriendlyError(err)})
}

func contentDisposition(fileName string) string {
	name := fileName
	if index := strings.LastIndex(strings.ReplaceAll(fileName, "\\", "/"), "/"); index != -1 {
		name = fileName[index+1:]
	}

	escaped := strings.ReplaceAll(name, `"`, "")
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, escaped, url.PathEscape(name))
}
