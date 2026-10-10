package files

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path string, data []byte, modTime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func TestGetFilesList(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR")

	// Completed browser upload with full metadata
	writeFile(t, filepath.Join(dir, "older"), []byte("hello"), now.Add(-time.Hour))
	writeFile(t, filepath.Join(dir, "older.info"), []byte(`{"Size":5,"MetaData":{"filename":"note.txt","filetype":"text/plain"}}`), now)

	// Completed CLI upload without a filetype: type comes from the extension
	writeFile(t, filepath.Join(dir, "newest"), []byte("not really a video"), now)
	writeFile(t, filepath.Join(dir, "newest.info"), []byte(`{"Size":18,"MetaData":{"filename":"clip.MP4"}}`), now)

	// No .info at all: type is sniffed from the content
	writeFile(t, filepath.Join(dir, "raw"), png, now.Add(-time.Minute))

	// Upload still in progress, plus its tusd lock file
	writeFile(t, filepath.Join(dir, "partial"), []byte("abc"), now.Add(time.Minute))
	writeFile(t, filepath.Join(dir, "partial.info"), []byte(`{"Size":100,"MetaData":{"filename":"big.jpg"}}`), now)
	writeFile(t, filepath.Join(dir, "partial.lock"), nil, now)

	// Upload whose length has not been declared yet
	writeFile(t, filepath.Join(dir, "deferred"), nil, now.Add(time.Minute))
	writeFile(t, filepath.Join(dir, "deferred.info"), []byte(`{"Size":0,"SizeIsDeferred":true,"MetaData":{"filename":"x.bin"}}`), now)

	got := New(dir, nil).getFilesList()

	want := []FileInfo{
		{ID: "newest", Name: "clip.MP4", Type: "video/mp4", Size: 18},
		{ID: "raw", Name: "raw", Type: "image/png", Size: int64(len(png))},
		{ID: "older", Name: "note.txt", Type: "text/plain", Size: 5},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d files %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.ID != w.ID || g.Name != w.Name || g.Type != w.Type || g.Size != w.Size {
			t.Errorf("file %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestHandleFileSetsType(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "abc"), []byte("data"), time.Now())
	writeFile(t, filepath.Join(dir, "abc.info"), []byte(`{"Size":4,"MetaData":{"filename":"photo.heic","filetype":"image/heic"}}`), time.Now())

	rec := httptest.NewRecorder()
	New(dir, nil).HandleFile(rec, httptest.NewRequest(http.MethodGet, "/api/files/abc", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/heic" {
		t.Errorf("Content-Type = %q, want image/heic", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="photo.heic"; filename*=UTF-8''photo.heic` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "private, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want finished files cached", cc)
	}
}

func TestHandleFileIgnoresBadType(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "abc"), []byte("data"), time.Now())
	writeFile(t, filepath.Join(dir, "abc.info"), []byte(`{"Size":4,"MetaData":{"filename":"a.png","filetype":"not a type"}}`), time.Now())

	rec := httptest.NewRecorder()
	New(dir, nil).HandleFile(rec, httptest.NewRequest(http.MethodGet, "/api/files/abc", nil))

	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png from the extension", ct)
	}
}

func TestHandleFileDoesNotCachePartialUpload(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "abc"), []byte("da"), time.Now())
	writeFile(t, filepath.Join(dir, "abc.info"), []byte(`{"Size":4,"MetaData":{"filename":"a.png"}}`), time.Now())

	rec := httptest.NewRecorder()
	New(dir, nil).HandleFile(rec, httptest.NewRequest(http.MethodGet, "/api/files/abc", nil))

	if cc := rec.Header().Get("Cache-Control"); cc != "" {
		t.Errorf("Cache-Control = %q, want none for an unfinished upload", cc)
	}
}
