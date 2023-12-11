package importer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"groundseg/docker"
	"groundseg/structs"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// checkExtension identifies the type of compressed file by its extension
func checkExtension(filename string) string {
	if strings.HasSuffix(filename, ".tar.gz") {
		return ".tar.gz"
	}
	return strings.ToLower(filepath.Ext(filename))
}

// extractZip extracts .zip files and sends % extracted to channel
func extractZip(src, dest string) error {
	// Open the zip archive
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Initialize total and extracted sizes
	var totalSize int64 = 0
	var extractedSize int64 = 0

	// Calculate the total size of all files in the zip
	for _, f := range r.File {
		totalSize += int64(f.UncompressedSize)
	}

	// Loop through the files in the zip archive
	for _, f := range r.File {
		// Open the file inside the zip
		rc, err := f.Open()
		if err != nil {
			return err
		}

		// Define the path and create directories as needed
		target := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
		} else {
			// Create the parent directory if it doesn't exist
			parent := filepath.Dir(target)
			if err := os.MkdirAll(parent, 0755); err != nil {
				return err
			}

			// Extract the file
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, f.Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, rc); err != nil {
				return err
			}
			file.Close()
		}
		rc.Close()

		// Update extracted size and send to the channel
		extractedSize += int64(f.UncompressedSize)
		percentExtracted := int(float64(extractedSize) / float64(totalSize) * 100)
		docker.ImportShipTransBus <- structs.UploadTransition{Type: "extracted", Value: percentExtracted}
	}
	return nil
}

// extractTarGz extracts .tar.gz files and sends % extracted to channel
func extractTarGz(src, dest string) error {
	// Open the tar.gz file
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Create a tar reader
	tr := tar.NewReader(gzr)

	// Initialize total and extracted sizes
	var totalSize int64 = 0
	var extractedSize int64 = 0

	// Loop through the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Update total size
		totalSize += header.Size

		// Define the path and create directories as needed
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// Extract the file
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				return err
			}
			file.Close()
		}

		// Update extracted size and send to the channel
		extractedSize += header.Size
		percentExtracted := int(float64(extractedSize) / float64(totalSize) * 100)
		docker.ImportShipTransBus <- structs.UploadTransition{Type: "extracted", Value: percentExtracted}
	}
	return nil
}
