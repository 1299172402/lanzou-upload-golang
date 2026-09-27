package lanzou

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	uploadURL = "https://pc.woozooo.com/html5up.php"
	// maxRetries 是上传失败后的最大尝试次数（含首次）。
	maxRetries = 10
	// connectTimeout 是单次上传请求的超时时间。
	connectTimeout = time.Hour
)

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

	contentType := w.FormDataContentType()
	// 缓存请求体，便于重试时重复发送。
	bodyBytes := buf.Bytes()

	client := &http.Client{
		Timeout: connectTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   connectTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: connectTimeout,
			ExpectContinueTimeout: time.Second,
		},
	}
	defer client.CloseIdleConnections()

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		body, err := sendUpload(client, bodyBytes, contentType, cookie)
		if err == nil {
			return body, nil
		}
		lastErr = err

		if attempt < maxRetries {
			// 指数退避，最长 30 秒。
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}
	}
	return "", fmt.Errorf("上传失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// sendUpload 执行一次上传请求。
func sendUpload(client *http.Client, body []byte, contentType, cookie string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://pc.woozooo.com")
	req.Header.Set("Referer", "https://pc.woozooo.com/mydisk.php?item=files&action=index")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36 Edg/152.0.0.0")
	req.Header.Set("Cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, respBody)
	}
	return string(respBody), nil
}
