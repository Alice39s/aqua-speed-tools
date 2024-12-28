package updater

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
	"github.com/ulikunitz/xz"
	"go.uber.org/zap"
)

// ArchiveReader defines the interface for archive readers.
type ArchiveReader interface {
	Next() (string, io.Reader, error)
	Close() error
}

type ReaderWithProgress struct {
	reader     io.Reader
	total      int64
	current    int64
	progressFn func(current, total int64)
}

func NewReaderWithProgress(reader io.Reader, total int64, fn func(current, total int64)) *ReaderWithProgress {
	return &ReaderWithProgress{
		reader:     reader,
		total:      total,
		progressFn: fn,
	}
}

func (r *ReaderWithProgress) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.current += int64(n)
		if r.progressFn != nil {
			r.progressFn(r.current, r.total)
		}
	}
	return n, err
}

// NewArchiveReader creates a new ArchiveReader based on the archive type.
func NewArchiveReader(path string, logger *zap.Logger) (ArchiveReader, error) {
	if strings.HasSuffix(path, ".zip") {
		return NewZipArchiveReader(path, logger)
	}
	return NewTarXzArchiveReader(path, logger)
}

type ZipArchiveReader struct {
	reader     *zip.ReadCloser
	files      []*zip.File
	index      int
	bufferPool sync.Pool
	logger     *zap.Logger
}

func NewZipArchiveReader(path string, logger *zap.Logger) (*ZipArchiveReader, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}

	return &ZipArchiveReader{
		reader: reader,
		files:  reader.File,
		index:  0,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bufio.NewReaderSize(nil, 32*1024) // 32KB buffer
			},
		},
		logger: logger,
	}, nil
}

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

	br := z.bufferPool.Get().(*bufio.Reader)
	br.Reset(rc)

	reader := &pooledReader{
		reader: br,
		closer: rc,
		pool:   &z.bufferPool,
	}

	progressBar := progressbar.DefaultBytes(
		int64(file.UncompressedSize64),
		fmt.Sprintf("Extracting %s", file.Name),
	)

	return file.Name, NewReaderWithProgress(reader, int64(file.UncompressedSize64),
		func(current, total int64) {
			progressBar.Set64(current)
			z.logger.Info("Extraction progress",
				zap.String("file", file.Name),
				zap.Int64("current", current),
				zap.Int64("total", total),
				zap.Float64("percentage", float64(current)/float64(total)*100))
		}), nil
}

func (z *ZipArchiveReader) Close() error {
	return z.reader.Close()
}

type TarXzArchiveReader struct {
	file      *os.File
	xzReader  io.Reader
	tarReader *tar.Reader
	logger    *zap.Logger
}

func NewTarXzArchiveReader(path string, logger *zap.Logger) (*TarXzArchiveReader, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAR.XZ file: %w", err)
	}

	// Enable read-ahead for better performance
	// syscall.Fadvise is not available on all platforms, so skip it for now.
	// syscall.Fadvise(int(f.Fd()), 0, 0, syscall.FADV_SEQUENTIAL)
	// TODO: use fadvise on linux

	// Optimize buffer size for small files
	bufferedReader := bufio.NewReaderSize(f, 256*1024) // 256KB buffer

	// Configure XZ reader for optimal small file performance
	xzConfig := xz.ReaderConfig{
		DictCap: 1024 * 1024, // 1MB dictionary
	}

	xzReader, err := xzConfig.NewReader(bufferedReader)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	progressReader := NewReaderWithProgress(xzReader, fi.Size(),
		func(current, total int64) {
			logger.Info("Decompression progress",
				zap.Int64("current", current),
				zap.Int64("total", total),
				zap.Float64("percentage", float64(current)/float64(total)*100))
		})

	tarReader := tar.NewReader(progressReader)

	return &TarXzArchiveReader{
		file:      f,
		xzReader:  xzReader,
		tarReader: tarReader,
		logger:    logger,
	}, nil
}

func (t *TarXzArchiveReader) Next() (string, io.Reader, error) {
	header, err := t.tarReader.Next()
	if err != nil {
		return "", nil, err
	}

	if header.Size > 0 {
		progressBar := progressbar.DefaultBytes(
			header.Size,
			fmt.Sprintf("Extracting %s", header.Name),
		)

		return header.Name, NewReaderWithProgress(t.tarReader, header.Size,
			func(current, total int64) {
				progressBar.Set64(current)
				t.logger.Debug("File extraction progress",
					zap.String("file", header.Name),
					zap.Int64("current", current),
					zap.Int64("total", total),
					zap.Float64("percentage", float64(current)/float64(total)*100))
			}), nil
	}

	return header.Name, t.tarReader, nil
}

func (t *TarXzArchiveReader) Close() error {
	if closer, ok := t.xzReader.(io.Closer); ok {
		closer.Close()
	}
	return t.file.Close()
}

type pooledReader struct {
	reader *bufio.Reader
	closer io.Closer
	pool   *sync.Pool
}

func (r *pooledReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *pooledReader) Close() error {
	r.pool.Put(r.reader)
	return r.closer.Close()
}
