package cbclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
)

type FileInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploadedAt"`
}

type Client struct {
	serverURL string
	token     string
	http      *http.Client
}

func NewClient(serverURL, token string) *Client {
	return &Client{
		serverURL: serverURL,
		token:     token,
		http:      &http.Client{},
	}
}

func (c *Client) GetClipboard() (string, error) {
	var result struct {
		Content string `json:"content"`
	}
	if err := c.get("/api/clipboard", &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

func (c *Client) SetClipboard(content string) error {
	return c.post("/api/clipboard", map[string]string{"content": content})
}

func (c *Client) ListFiles() ([]FileInfo, error) {
	var files []FileInfo
	if err := c.get("/api/files", &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *Client) DownloadFile(id string) (io.ReadCloser, int64, error) {
	path := "/api/files/" + id
	req, err := http.NewRequest("GET", c.serverURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("connecting to %s: %w", c.serverURL, err)
	}
	if resp.StatusCode == 401 {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("unauthorized — run 'cb login' first")
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("server error %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}
	size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	return resp.Body, size, nil
}

// UploadFileFromReader uploads file contents to the server using the tus creation-with-upload
// extension. size is the byte length of r; filename is used as the remote filename.
func (c *Client) UploadFileFromReader(r io.Reader, size int64, filename string) error {
	req, err := http.NewRequest("POST", c.serverURL+"/api/uploads", r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Length", strconv.FormatInt(size, 10))
	req.Header.Set("Content-Type", "application/offset+octet-stream")
	req.Header.Set("Upload-Offset", "0")
	req.Header.Set("Upload-Metadata", "filename "+base64.StdEncoding.EncodeToString([]byte(filepath.Base(filename))))
	req.ContentLength = size

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", c.serverURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized — run 'cb login' first")
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}
	return nil
}

func (c *Client) DeleteFile(id string) error {
	return c.delete("/api/files/" + id)
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest("GET", c.serverURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", c.serverURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized — run 'cb login' first")
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response from %s: %w", path, err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		preview := bytes.TrimSpace(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return fmt.Errorf("unexpected response from %s (expected JSON): %s", path, preview)
	}
	return nil
}

func (c *Client) post(path string, in any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.serverURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", c.serverURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized — run 'cb login' first")
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, bytes.TrimSpace(b))
	}
	return nil
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.serverURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", c.serverURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized — run 'cb login' first")
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}
	return nil
}
