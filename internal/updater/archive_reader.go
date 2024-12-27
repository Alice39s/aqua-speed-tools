package updater

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ulikunitz/xz"
)

// ArchiveReader defines the interface for archive readers.
type ArchiveReader interface {
	Next() (string, io.Reader, error)
	Close() error
}

// NewArchiveReader creates a new ArchiveReader based on the archive type.
func NewArchiveReader(path string) (ArchiveReader, error) {
	if strings.HasSuffix(path, ".zip") {
		return NewZipArchiveReader(path)
	}
	return NewTarXzArchiveReader(path)
}

// ZipArchiveReader implements ArchiveReader for ZIP files.
type ZipArchiveReader struct {
	reader *zip.ReadCloser
	files  []*zip.File
	index  int
}

// NewZipArchiveReader creates a new ZipArchiveReader.
func NewZipArchiveReader(path string) (*ZipArchiveReader, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	return &ZipArchiveReader{
		reader: reader,
		files:  reader.File,
		index:  0,
	}, nil
}

// Next returns the next file in the ZIP archive.
func (z *ZipArchiveReader) Next() (string, io.Reader, error) {
	if z.index >= len(z.files) {
		return "", nil, io.EOF
	}
	file := z.files[z.index]
	z.index++
	rc, err := file.Open()
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file %s: %w", file.Name, err)
	}
	return file.Name, rc, nil
}

// Close closes the ZIP archive.
func (z *ZipArchiveReader) Close() error {
	return z.reader.Close()
}

// TarXzArchiveReader implements ArchiveReader for TAR.XZ files.
type TarXzArchiveReader struct {
	file      *os.File
	tarReader *tar.Reader
}

// NewTarXzArchiveReader creates a new TarXzArchiveReader.
func NewTarXzArchiveReader(path string) (*TarXzArchiveReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAR.XZ file: %w", err)
	}

	xzReader, err := xz.NewReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}

	tarReader := tar.NewReader(xzReader)
	return &TarXzArchiveReader{
		file:      f,
		tarReader: tarReader,
	}, nil
}

// Next returns the next file in the TAR.XZ archive.
func (t *TarXzArchiveReader) Next() (string, io.Reader, error) {
	header, err := t.tarReader.Next()
	if err != nil {
		return "", nil, err
	}
	return header.Name, t.tarReader, nil
}

// Close closes the TAR.XZ archive.
func (t *TarXzArchiveReader) Close() error {
	return t.file.Close()
}
