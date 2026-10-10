package files

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func writeFiles(t *testing.T, dir string, files map[string]string, ages map[string]time.Duration) {
	t.Helper()
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
		mod := time.Now().Add(-ages[name])
		os.Chtimes(path, mod, mod)
	}
}

func TestGetFilesList(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"old": "hello", "old.info": `{"Size":5,"MetaData":{"filename":"note.txt","filetype":"text/plain"}}`,
		"cli": "not a video", "cli.info": `{"Size":11,"MetaData":{"filename":"clip.MP4"}}`, // type from extension
		"bad": "x", "bad.info": `{"Size":1,"MetaData":{"filename":"a.png","filetype":"not a type"}}`,
		"raw":     "\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR", // no .info: sniffed
		"partial": "abc", "partial.info": `{"Size":100}`, "partial.lock": "",
		"creating": "", // .info not written yet
		"deferred": "", "deferred.info": `{"SizeIsDeferred":true}`,
	}, map[string]time.Duration{"old": time.Hour, "bad": 2 * time.Hour, "raw": time.Minute})

	var got []string
	for _, f := range New(dir, nil).getFilesList() {
		got = append(got, f.ID+" "+f.Name+" "+f.Type)
	}
	want := []string{"cli clip.MP4 video/mp4", "raw raw image/png", "old note.txt text/plain", "bad a.png image/png"}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestDownloadHeaders(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"abc": "data", "abc.info": `{"Size":4,"MetaData":{"filename":"photo.heic","filetype":"image/heic"}}`,
	}, nil)
	rec := httptest.NewRecorder()
	New(dir, nil).HandleFile(rec, httptest.NewRequest("GET", "/api/files/abc", nil))

	for header, want := range map[string]string{
		"Content-Type":        "image/heic",
		"Content-Disposition": `attachment; filename="photo.heic"; filename*=UTF-8''photo.heic`,
		"Cache-Control":       "private, max-age=31536000, immutable",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
