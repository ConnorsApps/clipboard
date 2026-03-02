package files

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tus/tusd/v2/pkg/handler"
)

// Service manages file operations for a single user
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

// UserFiles holds the service and tusd handler for one user
type userFiles struct {
	service *Service
	tusd    *handler.Handler
}

// Manager holds per-user file services and tusd handlers
type Manager struct {
	baseDir              string
	broadcastForUser     func(userID string) func([]FileInfo)
	getUserIDFromRequest func(*http.Request) (string, bool)
	users                map[string]*userFiles
	mu                   sync.RWMutex
}

// NewManager creates a new files manager. broadcastForUser returns the broadcast callback for a given userID.
// getUserIDFromRequest extracts the authenticated user ID from the request (e.g. from context set by auth middleware).
func NewManager(baseDir string, broadcastForUser func(userID string) func([]FileInfo), getUserIDFromRequest func(*http.Request) (string, bool)) *Manager {
	return &Manager{
		baseDir:              baseDir,
		broadcastForUser:     broadcastForUser,
		getUserIDFromRequest: getUserIDFromRequest,
		users:                make(map[string]*userFiles),
	}
}

// GetOrCreate returns the service and tusd handler for the given user ID, creating them if needed
func (m *Manager) GetOrCreate(userID string) (*Service, *handler.Handler, error) {
	m.mu.RLock()
	uf, ok := m.users[userID]
	m.mu.RUnlock()
	if ok {
		return uf.service, uf.tusd, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if uf, ok = m.users[userID]; ok {
		return uf.service, uf.tusd, nil
	}
	userDir := filepath.Join(m.baseDir, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, nil, err
	}
	broadcastFunc := m.broadcastForUser(userID)
	svc := New(userDir, broadcastFunc)
	tusdHandler, err := NewTusdHandler(userDir, svc.BroadcastFilesList)
	if err != nil {
		return nil, nil, err
	}
	m.users[userID] = &userFiles{service: svc, tusd: tusdHandler}
	return svc, tusdHandler, nil
}

// HandleFile handles file download and delete, routing to the authenticated user's service
func (m *Manager) HandleFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.getUserIDFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	svc, _, err := m.GetOrCreate(userID)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Failed to get files service")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	svc.HandleFile(w, r)
}

// ListFiles handles listing files for the authenticated user
func (m *Manager) ListFiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.getUserIDFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	svc, _, err := m.GetOrCreate(userID)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Failed to get files service")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	svc.ListFiles(w, r)
}

// HandleUpload routes the request to the authenticated user's tusd handler.
// The tusd handler expects the path with the base prefix stripped (e.g. "" or "upload-id").
func (m *Manager) HandleUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.getUserIDFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_, tusdHandler, err := m.GetOrCreate(userID)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Failed to get tusd handler")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Strip /api/uploads/ or /api/uploads so tusd receives path "" or "upload-id"
	path := r.URL.Path
	if path == "/api/uploads" {
		r2 := *r
		r2.URL = cloneURL(r.URL)
		r2.URL.Path = "/"
		tusdHandler.ServeHTTP(w, &r2)
		return
	}
	if strings.HasPrefix(path, "/api/uploads/") {
		http.StripPrefix("/api/uploads/", tusdHandler).ServeHTTP(w, r)
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := *u
	return &u2
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
