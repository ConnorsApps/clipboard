package files

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Service manages file operations
type Service struct {
	filesDir      string
	broadcastFunc func([]FileInfo)
}

// New creates a new files service
func New(filesDir string, broadcastFunc func([]FileInfo)) *Service {
	return &Service{
		filesDir:      filesDir,
		broadcastFunc: broadcastFunc,
	}
}

// getOriginalFilename reads the original filename from the tusd .info file
func (s *Service) getOriginalFilename(fileID string) string {
	infoPath := filepath.Join(s.filesDir, fileID+".info")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		log.Warn().Err(err).Str("file", fileID).Msg("Failed to read .info file")
		return fileID
	}

	var info TusdInfo
	if err := json.Unmarshal(data, &info); err != nil {
		log.Warn().Err(err).Str("file", fileID).Msg("Failed to parse .info file")
		return fileID
	}

	if info.MetaData.Filename != "" {
		return info.MetaData.Filename
	}

	return fileID
}

// getFilesList returns the current list of files
func (s *Service) getFilesList() []FileInfo {
	files := []FileInfo{}

	// Read all files from the files directory
	entries, err := os.ReadDir(s.filesDir)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read files directory")
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip .info files created by tusd
		if strings.HasSuffix(entry.Name(), ".info") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			log.Error().Err(err).Str("file", entry.Name()).Msg("Failed to get file info")
			continue
		}

		// Get original filename from .info file
		originalName := s.getOriginalFilename(entry.Name())

		files = append(files, FileInfo{
			ID:         entry.Name(),
			Name:       originalName,
			Size:       info.Size(),
			UploadedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	return files
}

// BroadcastFilesList sends the current file list to all WebSocket clients
func (s *Service) BroadcastFilesList() {
	if s.broadcastFunc != nil {
		files := s.getFilesList()
		s.broadcastFunc(files)
	}
}

// HandleFile handles file download and delete operations
func (s *Service) HandleFile(w http.ResponseWriter, r *http.Request) {
	// Extract file ID from path
	fileID := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Sanitize file ID to prevent directory traversal
	fileID = filepath.Base(fileID)
	filePath := filepath.Join(s.filesDir, fileID)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Download file
		w.Header().Set("Content-Disposition", "attachment; filename=\""+fileID+"\"")
		http.ServeFile(w, r, filePath)

	case http.MethodDelete:
		// Delete file
		if err := os.Remove(filePath); err != nil {
			log.Error().Err(err).Str("file", fileID).Msg("Failed to delete file")
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}

		// Also delete .info file if it exists
		infoPath := filePath + ".info"
		if _, err := os.Stat(infoPath); err == nil {
			os.Remove(infoPath)
		}

		log.Info().Str("file", fileID).Msg("File deleted")
		
		// Broadcast updated file list to all WebSocket clients
		s.BroadcastFilesList()
		
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ListFiles handles listing all uploaded files
func (s *Service) ListFiles(w http.ResponseWriter, r *http.Request) {
	files := s.getFilesList()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
