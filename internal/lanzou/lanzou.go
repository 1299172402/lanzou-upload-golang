package lanzou

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const uploadURL = "https://pc.woozooo.com/html5up.php"

func Upload(filePath, folderID, cookie string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}

	fileName := filepath.Base(filePath)
	mimeType := mime.TypeByExtension(filepath.Ext(fileName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fields := [][2]string{
		{"task", "1"},
		{"vie", "2"},
		{"ve", "2"},
		{"id", "WU_FILE_0"},
		{"name", fileName},
		{"type", mimeType},
		{"lastModifiedDate", stat.ModTime().Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")},
		{"size", fmt.Sprintf("%d", stat.Size())},
		{"folder_id_bb_n", folderID},
	}
	for _, kv := range fields {
		if err := w.WriteField(kv[0], kv[1]); err != nil {
			return "", err
		}
	}

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="upload_file"; filename="%s"`, fileName)}
	h["Content-Type"] = []string{mimeType}
	part, err := w.CreatePart(h)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, uploadURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://pc.woozooo.com")
	req.Header.Set("Referer", "https://pc.woozooo.com/mydisk.php?item=files&action=index")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36 Edg/152.0.0.0")
	req.Header.Set("Cookie", cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return string(body), nil
}
