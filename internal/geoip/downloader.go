package geoip

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader manages retrieval and unpacking of open GeoIP MMDB datasets.
type Downloader struct {
	client *http.Client
}

// NewDownloader initializes an HTTP client for database downloads.
func NewDownloader() *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// DownloadDBIP retrieves the latest open-source DB-IP Lite MMDB database.
func (d *Downloader) DownloadDBIP(destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Current month year for DB-IP URL convention: https://download.db-ip.com/free/dbip-country-lite-YYYY-MM.mmdb.gz
	now := time.Now()
	filename := fmt.Sprintf("dbip-country-lite-%d-%02d.mmdb.gz", now.Year(), now.Month())
	url := fmt.Sprintf("https://download.db-ip.com/free/%s", filename)

	targetMMDB := filepath.Join(destDir, "dbip-country-lite.mmdb")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GeoGate-Downloader/1.0")

	resp, err := d.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Fallback to previous month if current month archive is not yet published
		prev := now.AddDate(0, -1, 0)
		filename = fmt.Sprintf("dbip-country-lite-%d-%02d.mmdb.gz", prev.Year(), prev.Month())
		url = fmt.Sprintf("https://download.db-ip.com/free/%s", filename)

		req2, _ := http.NewRequest(http.MethodGet, url, nil)
		req2.Header.Set("User-Agent", "GeoGate-Downloader/1.0")
		resp, err = d.client.Do(req2)
		if err != nil {
			return "", fmt.Errorf("failed to download database: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP error %d fetching %s", resp.StatusCode, url)
		}
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if strings.HasSuffix(url, ".gz") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to initialize gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	outFile, err := os.Create(targetMMDB)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return "", fmt.Errorf("failed to write MMDB file: %w", err)
	}

	return targetMMDB, nil
}
