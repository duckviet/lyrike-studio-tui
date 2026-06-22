package peaks

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ErrRangeNotSatisfiable is returned by ParseRangeHeader when the Range header
// is malformed, uses an unsupported unit, or asks for bytes outside the file.
var ErrRangeNotSatisfiable = errors.New("range not satisfiable")

// ParseRangeHeader parses an HTTP Range header of the form "bytes=start-end",
// "bytes=start-", or "bytes=-suffix" against the provided file size. Both
// start and end are inclusive byte indices.
func ParseRangeHeader(rangeHeader string, fileSize int64) (int64, int64, error) {
	if fileSize <= 0 {
		return 0, 0, ErrRangeNotSatisfiable
	}
	if rangeHeader == "" {
		return 0, 0, ErrRangeNotSatisfiable
	}

	const prefix = "bytes="
	if !strings.HasPrefix(rangeHeader, prefix) {
		return 0, 0, ErrRangeNotSatisfiable
	}
	rangeHeader = strings.TrimPrefix(rangeHeader, prefix)

	// Only a single byte range is supported.
	if strings.Contains(rangeHeader, ",") {
		return 0, 0, ErrRangeNotSatisfiable
	}

	dash := strings.Index(rangeHeader, "-")
	if dash < 0 {
		return 0, 0, ErrRangeNotSatisfiable
	}

	before := rangeHeader[:dash]
	after := rangeHeader[dash+1:]

	// Suffix range: bytes=-suffix
	if before == "" {
		if after == "" {
			return 0, 0, ErrRangeNotSatisfiable
		}
		suffix, err := strconv.ParseInt(after, 10, 64)
		if err != nil || suffix < 0 {
			return 0, 0, ErrRangeNotSatisfiable
		}
		start := fileSize - suffix
		if start < 0 {
			start = 0
		}
		return start, fileSize - 1, nil
	}

	start, err := strconv.ParseInt(before, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, ErrRangeNotSatisfiable
	}

	var end int64
	if after == "" {
		// Open-ended range: bytes=start-
		end = fileSize - 1
	} else {
		end, err = strconv.ParseInt(after, 10, 64)
		if err != nil || end < 0 {
			return 0, 0, ErrRangeNotSatisfiable
		}
	}

	if start >= fileSize || start > end {
		return 0, 0, ErrRangeNotSatisfiable
	}
	if end >= fileSize {
		end = fileSize - 1
	}

	return start, end, nil
}

// IterFileRange returns an io.Reader that yields the inclusive byte range
// [start, end] from path in 64 KiB chunks. If the file cannot be opened or the
// range is invalid, the reader returns the error on the first Read call.
func IterFileRange(path string, start, end int64) io.Reader {
	if start < 0 || end < 0 || start > end {
		return &errReader{err: fmt.Errorf("invalid file range %d-%d", start, end)}
	}

	f, err := os.Open(path)
	if err != nil {
		return &errReader{err: err}
	}

	if _, err := f.Seek(start, io.SeekStart); err != nil {
		_ = f.Close()
		return &errReader{err: err}
	}

	return &fileRangeReader{f: f, remaining: end - start + 1}
}

type fileRangeReader struct {
	f         *os.File
	remaining int64
}

func (r *fileRangeReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}

	chunkSize := int64(64 * 1024)
	if int64(len(p)) > chunkSize {
		p = p[:chunkSize]
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}

	n, err := r.f.Read(p)
	r.remaining -= int64(n)
	if err == io.EOF && r.remaining > 0 {
		return n, io.ErrUnexpectedEOF
	}
	if r.remaining == 0 && err == nil {
		err = io.EOF
		_ = r.f.Close()
	}
	return n, err
}

type errReader struct {
	err error
}

func (r *errReader) Read([]byte) (int, error) {
	return 0, r.err
}
