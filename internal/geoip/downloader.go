package geoip

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sguesdon/geoip-proxy-filter/internal/config"
)

type Downloader struct {
    config *config.Config
}

func NewDownloader(config *config.Config) *Downloader {
    return &Downloader{
        config: config,
    }
}

func (d *Downloader) UpdateDatabase() error {
    if d.config.MaxMind.AccountID == "" || d.config.MaxMind.LicenseKey == "" {
        return fmt.Errorf("GEOLITE_ACCOUNT_ID and GEOLITE_LICENCE_KEY environment variables must be set")
    }

    // Download the tar.gz file
    url := fmt.Sprintf(d.config.MaxMind.URL, d.config.MaxMind.LicenseKey)
    tarFilename := d.config.TmpDir + d.config.MaxMind.GeolitePrefix + "_latest.tar.gz"

    if err := d.downloadFile(url, tarFilename); err != nil {
        return fmt.Errorf("failed to download file: %w", err)
    }
    defer os.Remove(tarFilename)

    // Extract the .mmdb file
    mmdbFilename := d.config.MaxMind.GeolitePrefix + ".mmdb"
    if err := d.extractMMDBFile(tarFilename, mmdbFilename); err != nil {
        return fmt.Errorf("failed to extract file: %w", err)
    }

    if err := os.Rename(mmdbFilename, d.config.GeoIP.DatabasePath); err != nil {
        return fmt.Errorf("failed to move file to location: %w", err)
    }

    return nil
}

func (d *Downloader) downloadFile(url, filename string) error {
    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }

    req.SetBasicAuth(d.config.MaxMind.AccountID, d.config.MaxMind.LicenseKey)

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("bad status: %s", resp.Status)
    }

    out, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}

func (d *Downloader) extractMMDBFile(tarFilename, mmdbFilename string) error {
    file, err := os.Open(tarFilename)
    if err != nil {
        return err
    }
    defer file.Close()

    gzr, err := gzip.NewReader(file)
    if err != nil {
        return err
    }
    defer gzr.Close()

    tr := tar.NewReader(gzr)

    for {
        header, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        if strings.HasSuffix(header.Name, ".mmdb") {
            outFile, err := os.Create(mmdbFilename)
            if err != nil {
                return err
            }
            defer outFile.Close()

            _, err = io.Copy(outFile, tr)
            return err
        }
    }

    return fmt.Errorf("no .mmdb file found in archive")
}