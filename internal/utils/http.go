package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/nevcea-sub/minecraft-server-launcher/internal/logger"
	"github.com/schollz/progressbar/v3"
)

const (
	UserAgent       = "minecraft-server-launcher/1.0"
	DefaultTimeout  = 30 * time.Second
	DownloadBufSize = 128 * 1024
	MaxRetries      = 3
	RetryDelay      = 2 * time.Second
	RetryBackoff    = 2.0
)

var HTTPClient = &http.Client{
	Timeout: DefaultTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

func DoRequest(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	if client == nil {
		client = HTTPClient
	}

	var lastErr error
	delay := RetryDelay

	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * RetryBackoff)
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", UserAgent)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < MaxRetries-1 {
				logger.Warn("Request failed (attempt %d/%d), retrying: %v", attempt+1, MaxRetries, err)
			}
			continue
		}

		if resp.StatusCode == 200 {
			return resp, nil
		}

		lastErr = fmt.Errorf("API returned status %d", resp.StatusCode)
		resp.Body.Close()

		if attempt < MaxRetries-1 {
			logger.Warn("Request failed (attempt %d/%d), retrying: status %d", attempt+1, MaxRetries, resp.StatusCode)
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", MaxRetries, lastErr)
}

func DownloadFile(ctx context.Context, url, filename string) error {
	resp, err := DoRequest(ctx, HTTPClient, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempFile := filename + ".part"
	if _, err := os.Stat(tempFile); err == nil {
		logger.Info("Found incomplete download, removing: %s", tempFile)
		if err := os.Remove(tempFile); err != nil {
			logger.Warn("Failed to remove incomplete download: %v", err)
		}
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	closed := false
	defer func() {
		if !closed {
			if err := out.Close(); err != nil {
				logger.Warn("Failed to close temp file: %v", err)
			}
			if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
				logger.Warn("Failed to remove temp file: %v", err)
			}
		}
	}()

	var bar *progressbar.ProgressBar
	if resp.ContentLength > 0 {
		bar = progressbar.NewOptions64(
			resp.ContentLength,
			progressbar.OptionSetWriter(os.Stdout),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionSetDescription("Downloading"),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stdout, "\n")
			}),
		)
	} else {
		bar = progressbar.NewOptions64(
			-1,
			progressbar.OptionSetWriter(os.Stdout),
			progressbar.OptionSetDescription("Downloading"),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stdout, "\n")
			}),
		)
	}

	buf := make([]byte, DownloadBufSize)
	var writer io.Writer = out
	if bar != nil {
		writer = io.MultiWriter(out, bar)
	}

	done := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(writer, resp.Body, buf)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	if bar != nil {
		bar.Close()
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	closed = true

	if _, err := os.Stat(filename); err == nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	logger.Info("Download complete!")

	return nil
}
