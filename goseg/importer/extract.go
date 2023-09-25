package importer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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

// extractZip extracts .zip files
func extractZip(src, dest string) error {
	// Open the zip archive
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Iterate through each file
	for _, f := range r.File {
		// Create full path for the file
		fpath := filepath.Join(dest, f.Name)

		// Create all directories in the path
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// Skip if it's a directory entry
		if f.FileInfo().IsDir() {
			continue
		}

		// Open the file from the zip archive
		rc, err := f.Open()
		if err != nil {
			return err
		}

		// Create the destination file
		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, rc)

		// Close the file handles
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
func extractTarGz(src, dest string) error {
	// Open the compressed file
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Iterate through each file
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Create full path for the file
		fpath := filepath.Join(dest, header.Name)

		// Create all directories in the path
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// Skip if it's a directory entry
		if header.FileInfo().IsDir() {
			continue
		}

		// Create the destination file
		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, tr)

		// Close the file handle
		outFile.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
