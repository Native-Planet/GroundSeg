package importer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"groundseg/docker"
	"groundseg/structs"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func checkExtension(filename string) string {
	if strings.HasSuffix(filename, ".tar.gz") {
		return ".tar.gz"
	}
	return strings.ToLower(filepath.Ext(filename))
}

func extractZip(src, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	var totalSize int64 = 0
	var extractedSize int64 = 0
	var fileCount int = 0
	var processedCount int = 0

	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			totalSize += int64(f.UncompressedSize64)
			fileCount++
		}
	}

	buffer := make([]byte, 1024*1024)

	for _, f := range r.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if strings.Contains(f.Name, "__MACOSX") ||
				filepath.Base(f.Name) == ".DS_Store" ||
				filepath.Base(f.Name) == "conn.sock" {
				continue
			}

			target := filepath.Join(dest, f.Name)

			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(target, f.Mode()); err != nil {
					return err
				}
				continue
			}

			parent := filepath.Dir(target)
			if err := os.MkdirAll(parent, 0755); err != nil {
				return err
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, f.Mode())
			if err != nil {
				rc.Close()
				return err
			}

			written, err := io.CopyBuffer(file, rc, buffer)

			rc.Close()
			file.Close()

			if err != nil {
				return err
			}

			extractedSize += written
			processedCount++

			percentExtracted := int((float64(processedCount) / float64(fileCount) * 50) +
				(float64(extractedSize) / float64(totalSize) * 50))
			if percentExtracted > 99 {
				percentExtracted = 99
			}

			docker.ImportShipTransBus <- structs.UploadTransition{
				Type:  "extracted",
				Value: percentExtracted,
			}
		}
	}

	// Send explicit 100% when complete
	docker.ImportShipTransBus <- structs.UploadTransition{
		Type:  "extracted",
		Value: 100,
	}
	return nil
}

func extractTarGz(src, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	file, err := os.Open(src)
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
	buffer := make([]byte, 1024*1024) // Increase to 1MB

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	totalSize := fileInfo.Size()
	var processedBytes int64 = 0
	lastUpdate := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			header, err := tr.Next()
			if err == io.EOF {
				// Send explicit 100% when complete
				docker.ImportShipTransBus <- structs.UploadTransition{
					Type:  "extracted",
					Value: 100,
				}
				return nil
			}
			if err != nil {
				return err
			}

			if strings.Contains(header.Name, "__MACOSX") ||
				filepath.Base(header.Name) == ".DS_Store" ||
				filepath.Base(header.Name) == "conn.sock" {
				continue
			}

			target := filepath.Join(dest, header.Name)

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}
			case tar.TypeReg:
				parent := filepath.Dir(target)
				if err := os.MkdirAll(parent, 0755); err != nil {
					return err
				}

				destFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}

				written, err := io.CopyBuffer(destFile, tr, buffer)
				destFile.Close()

				if err != nil {
					return err
				}

				processedBytes += written

				if time.Since(lastUpdate) > time.Second {
					percentExtracted := int(float64(processedBytes) / float64(totalSize*2) * 100)
					if percentExtracted > 99 {
						percentExtracted = 99
					}

					docker.ImportShipTransBus <- structs.UploadTransition{
						Type:  "extracted",
						Value: percentExtracted,
					}
					lastUpdate = time.Now()
				}
			}
		}
	}
}

func extractTar(src, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	totalSize := fileInfo.Size()
	tr := tar.NewReader(file)

	buffer := make([]byte, 1024*1024) // Increase to 1MB

	var processedBytes int64 = 0
	lastUpdate := time.Now()
	var fileCount int = 0

	// First count files
	tmpFile, _ := os.Open(src)
	defer tmpFile.Close()
	tmpTr := tar.NewReader(tmpFile)
	for {
		_, err := tmpTr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		fileCount++
	}

	var processedCount int = 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			header, err := tr.Next()
			if err == io.EOF {
				// Send explicit 100% when complete
				docker.ImportShipTransBus <- structs.UploadTransition{
					Type:  "extracted",
					Value: 100,
				}
				return nil
			}
			if err != nil {
				return err
			}

			processedCount++

			if strings.Contains(header.Name, "__MACOSX") ||
				filepath.Base(header.Name) == ".DS_Store" ||
				filepath.Base(header.Name) == "conn.sock" {
				continue
			}

			target := filepath.Join(dest, header.Name)

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}
			case tar.TypeReg:
				parent := filepath.Dir(target)
				if err := os.MkdirAll(parent, 0755); err != nil {
					return err
				}

				destFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}

				written, err := io.CopyBuffer(destFile, tr, buffer)
				destFile.Close()

				if err != nil {
					return err
				}

				processedBytes += written

				// Only update progress at most once per second
				if time.Since(lastUpdate) > time.Second {
					// Use both file count and processed bytes for progress
					percentExtracted := int((float64(processedCount) / float64(fileCount) * 50) +
						(float64(processedBytes) / float64(totalSize) * 50))
					if percentExtracted > 99 {
						percentExtracted = 99
					}

					docker.ImportShipTransBus <- structs.UploadTransition{
						Type:  "extracted",
						Value: percentExtracted,
					}
					lastUpdate = time.Now()
				}
			}
		}
	}
}
