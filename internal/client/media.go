package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// mediaChunkSize is exactly 5 MiB (correction #10).
	mediaChunkSize = 5 * 1024 * 1024
	// mediaMaxPolls is the maximum number of STATUS polls (correction #10).
	mediaMaxPolls = 20
)

// UploadMedia uploads a media file and returns its media_id_string.
// altText is only applied for image/* MIME types (correction #10).
func (c *Client) UploadMedia(ctx context.Context, data []byte, mimeType, altText string) (string, error) {
	size := len(data)

	// INIT phase.
	mediaID, err := c.mediaInit(ctx, size, mimeType)
	if err != nil {
		return "", fmt.Errorf("media INIT: %w", err)
	}

	// APPEND phase — 5 MiB chunks.
	for i := 0; i*mediaChunkSize < size; i++ {
		start := i * mediaChunkSize
		end := start + mediaChunkSize
		if end > size {
			end = size
		}
		if err := c.mediaAppend(ctx, mediaID, i, data[start:end]); err != nil {
			return "", fmt.Errorf("media APPEND chunk %d: %w", i, err)
		}
	}

	// FINALIZE phase.
	if err := c.mediaFinalize(ctx, mediaID); err != nil {
		return "", fmt.Errorf("media FINALIZE: %w", err)
	}

	// STATUS polling.
	if err := c.mediaPollStatus(ctx, mediaID); err != nil {
		return "", fmt.Errorf("media STATUS: %w", err)
	}

	// Alt text — only for image/* (correction #10).
	if altText != "" && strings.HasPrefix(mimeType, "image/") {
		if err := c.mediaSetAltText(ctx, mediaID, altText); err != nil {
			return "", fmt.Errorf("media alt text: %w", err)
		}
	}

	return mediaID, nil
}

func (c *Client) mediaInit(ctx context.Context, totalBytes int, mimeType string) (string, error) {
	params := url.Values{}
	params.Set("command", "INIT")
	params.Set("total_bytes", strconv.Itoa(totalBytes))
	params.Set("media_type", mimeType)

	body, err := c.doPOSTForm(ctx, MediaUploadURL, c.getUploadHeaders(), params.Encode())
	if err != nil {
		return "", err
	}
	var resp struct {
		MediaIDString string `json:"media_id_string"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.MediaIDString == "" {
		return "", fmt.Errorf("media init: empty media_id_string in response")
	}
	return resp.MediaIDString, nil
}

func (c *Client) mediaAppend(ctx context.Context, mediaID string, segmentIndex int, chunk []byte) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("command", "APPEND"); err != nil {
		return fmt.Errorf("APPEND: write command: %w", err)
	}
	if err := mw.WriteField("media_id", mediaID); err != nil {
		return fmt.Errorf("APPEND: write media ID: %w", err)
	}
	if err := mw.WriteField("segment_index", strconv.Itoa(segmentIndex)); err != nil {
		return fmt.Errorf("APPEND: write segment index: %w", err)
	}
	fw, err := mw.CreateFormField("media")
	if err != nil {
		return err
	}
	if _, err := fw.Write(chunk); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("APPEND: close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, MediaUploadURL, &buf)
	if err != nil {
		return err
	}
	h := c.getUploadHeaders()
	for k, vs := range h {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("content-type", mw.FormDataContentType())

	_, doErr := c.do(req)
	if doErr != nil {
		return doErr
	}
	return nil
}

func (c *Client) mediaFinalize(ctx context.Context, mediaID string) error {
	params := url.Values{}
	params.Set("command", "FINALIZE")
	params.Set("media_id", mediaID)
	_, err := c.doPOSTForm(ctx, MediaUploadURL, c.getUploadHeaders(), params.Encode())
	return err
}

func (c *Client) mediaPollStatus(ctx context.Context, mediaID string) error {
	for i := 0; i < mediaMaxPolls; i++ {
		statusURL := MediaUploadURL + "?command=STATUS&media_id=" + mediaID
		body, err := c.doGET(ctx, statusURL, c.getUploadHeaders())
		if err != nil {
			return err
		}
		var resp struct {
			ProcessingInfo *struct {
				State          string `json:"state"`
				CheckAfterSecs *int   `json:"check_after_secs"`
			} `json:"processing_info"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return err
		}
		if resp.ProcessingInfo == nil {
			return nil
		}
		switch resp.ProcessingInfo.State {
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("media processing failed")
		}
		delay := 2 * time.Second
		if resp.ProcessingInfo.CheckAfterSecs != nil {
			d := time.Duration(*resp.ProcessingInfo.CheckAfterSecs) * time.Second
			if d >= time.Second {
				delay = d
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("media processing timed out after %d polls", mediaMaxPolls)
}

func (c *Client) mediaSetAltText(ctx context.Context, mediaID, altText string) error {
	body := map[string]any{
		"media_id": mediaID,
		"alt_text": map[string]string{"text": altText},
	}
	h := c.getJSONHeaders()
	_, err := c.doPOSTJSON(ctx, MediaMetadataURL, h, body)
	return err
}
