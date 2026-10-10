package files

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
	// Serializes list broadcasts so an older list can't arrive after a newer one
	broadcastMu sync.Mutex
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

// FilesForUser returns the current file list for the given user
func (m *Manager) FilesForUser(userID string) []FileInfo {
	svc, _, err := m.GetOrCreate(userID)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Failed to get files service")
		return []FileInfo{}
	}
	return svc.getFilesList()
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

// Common media missing from Go's built-in MIME table (Alpine has no /etc/mime.types)
var extraTypes = map[string]string{
	".heic": "image/heic",
	".heif": "image/heif",
	".mov":  "video/quicktime",
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".webm": "video/webm",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".txt":  "text/plain; charset=utf-8",
}

type fileMeta struct {
	name     string
	mimeType string
	complete bool
}

// readMeta returns a stored upload's original name, MIME type and completion state.
func (s *Service) readMeta(fileID string, stat os.FileInfo) fileMeta {
	// Without readable .info, a fresh empty file is an upload tusd is still creating
	settled := stat.Size() > 0 || time.Since(stat.ModTime()) > time.Minute
	meta := fileMeta{name: fileID, complete: settled}

	infoPath := filepath.Join(s.filesDir, fileID+".info")
	if data, err := os.ReadFile(infoPath); err != nil {
		if !os.IsNotExist(err) {
			log.Warn().Err(err).Str("file", fileID).Msg("Failed to read .info file")
		}
	} else {
		var info TusdInfo
		if err := json.Unmarshal(data, &info); err != nil {
			if settled {
				log.Warn().Err(err).Str("file", fileID).Msg("Failed to parse .info file")
			}
		} else {
			if info.MetaData.Filename != "" {
				meta.name = info.MetaData.Filename
			}
			// Client-supplied; use only if valid
			if _, _, err := mime.ParseMediaType(info.MetaData.Filetype); err == nil {
				meta.mimeType = info.MetaData.Filetype
			}
			// tusd appends chunks to an initially empty file
			meta.complete = !info.SizeIsDeferred && stat.Size() >= info.Size
		}
	}

	if meta.mimeType == "" {
		meta.mimeType = detectType(filepath.Join(s.filesDir, fileID), meta.name)
	}
	return meta
}

// detectType guesses a MIME type from the extension, then by sniffing
func detectType(path, name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	if t, ok := extraTypes[ext]; ok {
		return t
	}

	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := io.ReadFull(f, buf)
	return http.DetectContentType(buf[:n])
}

// getFilesList returns the completed uploads, newest first
func (s *Service) getFilesList() []FileInfo {
	files := []FileInfo{}

	// Read all files from the files directory
	entries, err := os.ReadDir(s.filesDir)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read files directory")
		return files
	}

	modTimes := make(map[string]time.Time, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip tusd's metadata and lock files
		if strings.HasSuffix(entry.Name(), ".info") || strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			log.Error().Err(err).Str("file", entry.Name()).Msg("Failed to get file info")
			continue
		}

		meta := s.readMeta(entry.Name(), info)
		if !meta.complete {
			continue
		}

		modTimes[entry.Name()] = info.ModTime()
		files = append(files, FileInfo{
			ID:         entry.Name(),
			Name:       meta.name,
			Type:       meta.mimeType,
			Size:       info.Size(),
			UploadedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	sort.SliceStable(files, func(i, j int) bool {
		return modTimes[files[i].ID].After(modTimes[files[j].ID])
	})

	return files
}

// BroadcastFilesList sends the current file list to all WebSocket clients
func (s *Service) BroadcastFilesList() {
	if s.broadcastFunc == nil {
		return
	}
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	s.broadcastFunc(s.getFilesList())
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
	stat, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Download file with the original filename, not the on-disk storage ID
		meta := s.readMeta(fileID, stat)
		originalName := meta.name
		asciiName := strings.Map(func(r rune) rune {
			if r < 0x20 || r == '"' || r > 0x7e {
				return '_'
			}
			return r
		}, originalName)
		w.Header().Set("Content-Disposition", "attachment; filename=\""+asciiName+"\"; filename*=UTF-8''"+url.QueryEscape(originalName))
		// Stored files have no extension to infer it from
		w.Header().Set("Content-Type", meta.mimeType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if meta.complete {
			// Upload IDs are never reused, so finished files never change
			w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		}
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
