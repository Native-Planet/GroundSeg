package importer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"groundseg/docker/events"
	"groundseg/structs"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	archiveExtractionTimeout   = 4 * time.Hour
	extractionProgressInterval = time.Second
	extractionBufferSize       = 1024 * 1024
)

type archiveEntryKind int

const (
	archiveEntryDirectory archiveEntryKind = iota
	archiveEntryFile
	archiveEntryOther
)

type archiveEntry struct {
	name       string
	mode       os.FileMode
	size       int64
	kind       archiveEntryKind
	reader     io.Reader
	closeEntry func() error
}

type archiveStats struct {
	fileCount  int64
	totalBytes int64
}

type archiveIterator interface {
	Next() (*archiveEntry, error)
	Close() error
}

type archiveExtractionStrategy interface {
	Open(src string) (archiveIterator, archiveStats, error)
	ShouldCountEntry(entry *archiveEntry, skipped bool) bool
	ShouldWriteEntry(entry *archiveEntry) bool
}

type zipArchiveStrategy struct{}
type tarArchiveStrategy struct {
	compressed bool
}

func checkExtension(filename string) string {
	if strings.HasSuffix(filename, ".tar.gz") {
		return ".tar.gz"
	}
	return strings.ToLower(filepath.Ext(filename))
}

func shouldIgnoreArchiveEntry(name string) bool {
	return strings.Contains(name, "__MACOSX") || filepath.Base(name) == ".DS_Store" || filepath.Base(name) == "conn.sock"
}

func archiveOutputPath(dest, name string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("archive entry has absolute path: %s", name)
	}
	if cleanName == "." || cleanName == string(filepath.Separator) {
		return "", fmt.Errorf("archive entry has invalid path: %s", name)
	}
	if strings.HasPrefix(cleanName, "..") && (cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator))) {
		return "", fmt.Errorf("archive entry has parent traversal path: %s", name)
	}
	return filepath.Join(dest, cleanName), nil
}

func extractionStrategyForFilename(filename string) (archiveExtractionStrategy, error) {
	switch checkExtension(filename) {
	case ".zip":
		return &zipArchiveStrategy{}, nil
	case ".tar.gz", ".tgz":
		return &tarArchiveStrategy{compressed: true}, nil
	case ".tar":
		return &tarArchiveStrategy{compressed: false}, nil
	default:
		return nil, fmt.Errorf("unsupported file type %v", filename)
	}
}

func extractUploadedArchive(src, dest, filename string) error {
	strategy, err := extractionStrategyForFilename(filename)
	if err != nil {
		return err
	}
	return extractWithStrategy(src, dest, strategy)
}

func extractWithStrategy(src, dest string, strategy archiveExtractionStrategy) error {
	ctx, cancel := context.WithTimeout(context.Background(), archiveExtractionTimeout)
	defer cancel()

	iterator, stats, err := strategy.Open(src)
	if err != nil {
		return err
	}
	defer iterator.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	progress := &extractionProgressTracker{
		fileCount:  stats.fileCount,
		totalBytes: stats.totalBytes,
		last:       time.Now(),
	}
	processedCount := int64(0)
	processedBytes := int64(0)
	buffer := make([]byte, extractionBufferSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		entry, err := iterator.Next()
		if err == io.EOF {
			events.DefaultEventRuntime().PublishImportShipTransition(context.Background(), structs.UploadTransition{Type: "extracted", Value: 100})
			return nil
		}
		if err != nil {
			return err
		}

		skipped := shouldIgnoreArchiveEntry(entry.name)
		if strategy.ShouldCountEntry(entry, skipped) {
			processedCount++
		}
		if skipped {
			if entry.closeEntry != nil {
				if err := entry.closeEntry(); err != nil {
					return err
				}
			}
			continue
		}

		target, err := archiveOutputPath(dest, entry.name)
		if err != nil {
			if entry.closeEntry != nil {
				if err := entry.closeEntry(); err != nil {
					return err
				}
			}
			return err
		}

		switch entry.kind {
		case archiveEntryDirectory:
			if err := os.MkdirAll(target, entry.mode); err != nil {
				if entry.closeEntry != nil {
					entry.closeEntry()
				}
				return err
			}
		case archiveEntryFile:
			if !strategy.ShouldWriteEntry(entry) {
				if entry.closeEntry != nil {
					if err := entry.closeEntry(); err != nil {
						return err
					}
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				if entry.closeEntry != nil {
					entry.closeEntry()
				}
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, entry.mode)
			if err != nil {
				if entry.closeEntry != nil {
					entry.closeEntry()
				}
				return err
			}

			written, err := io.CopyBuffer(file, entry.reader, buffer)
			closeErr := file.Close()
			if entry.closeEntry != nil {
				closeReaderErr := entry.closeEntry()
				if closeReaderErr != nil {
					return closeReaderErr
				}
			}
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}

			processedBytes += written
			progress.tryPublish(processedCount, processedBytes)
		case archiveEntryOther:
			if entry.closeEntry != nil {
				if err := entry.closeEntry(); err != nil {
					return err
				}
			}
		}
	}
}

func (s *zipArchiveStrategy) Open(src string) (archiveIterator, archiveStats, error) {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return nil, archiveStats{}, err
	}
	stats := archiveStats{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		stats.fileCount++
		stats.totalBytes += file.FileInfo().Size()
	}
	return &zipArchiveIterator{
		reader: reader,
		files:  reader.File,
	}, stats, nil
}

func (s *zipArchiveStrategy) ShouldCountEntry(entry *archiveEntry, skipped bool) bool {
	return !skipped && entry.kind == archiveEntryFile
}

func (s *zipArchiveStrategy) ShouldWriteEntry(entry *archiveEntry) bool {
	return entry.kind == archiveEntryFile
}

func (s *tarArchiveStrategy) Open(src string) (archiveIterator, archiveStats, error) {
	fileInfo, err := os.Stat(src)
	if err != nil {
		return nil, archiveStats{}, err
	}
	iterator, err := newTarArchiveIterator(src, s.compressed)
	if err != nil {
		return nil, archiveStats{}, err
	}

	iteratorForCount, err := newTarArchiveIterator(src, s.compressed)
	if err != nil {
		iterator.Close()
		return nil, archiveStats{}, err
	}
	defer iteratorForCount.Close()

	stats := archiveStats{fileCount: 0, totalBytes: fileInfo.Size()}
	for {
		entry, err := iteratorForCount.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			iterator.Close()
			return nil, archiveStats{}, err
		}
		if entry != nil {
			stats.fileCount++
		}
	}

	return iterator, stats, nil
}

func (s *tarArchiveStrategy) ShouldCountEntry(entry *archiveEntry, skipped bool) bool {
	return true
}

func (s *tarArchiveStrategy) ShouldWriteEntry(entry *archiveEntry) bool {
	return entry.kind == archiveEntryFile
}

type zipArchiveIterator struct {
	reader *zip.ReadCloser
	files  []*zip.File
	idx    int
}

func (z *zipArchiveIterator) Next() (*archiveEntry, error) {
	if z.idx >= len(z.files) {
		return nil, io.EOF
	}
	file := z.files[z.idx]
	z.idx++

	info := file.FileInfo()
	entry := &archiveEntry{
		name: file.Name,
		mode: info.Mode(),
		size: info.Size(),
	}
	if info.IsDir() {
		entry.kind = archiveEntryDirectory
		return entry, nil
	}
	entry.kind = archiveEntryFile
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	entry.reader = f
	entry.closeEntry = f.Close
	return entry, nil
}

func (z *zipArchiveIterator) Close() error {
	return z.reader.Close()
}

type tarArchiveIterator struct {
	reader io.Reader
	closer io.Closer
	tr     *tar.Reader
}

func newTarArchiveIterator(src string, compressed bool) (*tarArchiveIterator, error) {
	file, err := os.Open(src)
	if err != nil {
		return nil, err
	}

	reader := io.Reader(file)
	var closer io.Closer = file
	if compressed {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, err
		}
		reader = gzReader
		closer = combinedCloser(gzReader, file)
	}

	return &tarArchiveIterator{
		reader: reader,
		closer: closer,
		tr:     tar.NewReader(reader),
	}, nil
}

func combinedCloser(closers ...io.Closer) io.Closer {
	return ioCloserFunc(func() error {
		var firstError error
		for _, closer := range closers {
			if closer == nil {
				continue
			}
			if err := closer.Close(); err != nil && firstError == nil {
				firstError = err
			}
		}
		return firstError
	})
}

func (t *tarArchiveIterator) Next() (*archiveEntry, error) {
	header, err := t.tr.Next()
	if err != nil {
		return nil, err
	}

	entry := &archiveEntry{
		name: header.Name,
		mode: os.FileMode(header.Mode),
		size: header.Size,
	}

	switch header.Typeflag {
	case tar.TypeDir:
		entry.kind = archiveEntryDirectory
	case tar.TypeReg, tar.TypeRegA:
		entry.kind = archiveEntryFile
		entry.reader = io.LimitReader(t.tr, header.Size)
	default:
		entry.kind = archiveEntryOther
	}

	if entry.kind == archiveEntryFile {
		entry.closeEntry = nil
	}
	return entry, nil
}

func (t *tarArchiveIterator) Close() error {
	if t.closer == nil {
		return nil
	}
	return t.closer.Close()
}

type extractionProgressTracker struct {
	fileCount  int64
	totalBytes int64
	last       time.Time
}

func (p *extractionProgressTracker) tryPublish(processedEntries, processedBytes int64) {
	if p.totalBytes == 0 || p.fileCount == 0 {
		return
	}
	if time.Since(p.last) <= extractionProgressInterval {
		return
	}
	percentExtracted := int(float64(processedEntries)/float64(p.fileCount)*50 + float64(processedBytes)/float64(p.totalBytes)*50)
	if percentExtracted > 99 {
		percentExtracted = 99
	}
	events.DefaultEventRuntime().PublishImportShipTransition(context.Background(), structs.UploadTransition{
		Type:  "extracted",
		Value: percentExtracted,
	})
	p.last = time.Now()
}

func extractZip(src, dest string) error {
	return extractWithStrategy(src, dest, &zipArchiveStrategy{})
}

func extractTarGz(src, dest string) error {
	return extractWithStrategy(src, dest, &tarArchiveStrategy{compressed: true})
}

func extractTar(src, dest string) error {
	return extractWithStrategy(src, dest, &tarArchiveStrategy{compressed: false})
}

type ioCloserFunc func() error

func (fn ioCloserFunc) Close() error {
	return fn()
}
