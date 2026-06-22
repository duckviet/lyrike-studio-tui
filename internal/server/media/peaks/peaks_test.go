package peaks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func f32leBytes(samples ...float32) []byte {
	buf := new(bytes.Buffer)
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}

func TestParseRangeHeader_Normal(t *testing.T) {
	start, end, err := ParseRangeHeader("bytes=0-499", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 0 || end != 499 {
		t.Fatalf("expected start=0 end=499, got start=%d end=%d", start, end)
	}
}

func TestParseRangeHeader_OpenEnded(t *testing.T) {
	start, end, err := ParseRangeHeader("bytes=500-", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 500 || end != 999 {
		t.Fatalf("expected start=500 end=999, got start=%d end=%d", start, end)
	}
}

func TestParseRangeHeader_Suffix(t *testing.T) {
	start, end, err := ParseRangeHeader("bytes=-500", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 500 || end != 999 {
		t.Fatalf("expected start=500 end=999, got start=%d end=%d", start, end)
	}
}

func TestParseRangeHeader_SuffixLargerThanFile(t *testing.T) {
	start, end, err := ParseRangeHeader("bytes=-1500", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 0 || end != 999 {
		t.Fatalf("expected start=0 end=999, got start=%d end=%d", start, end)
	}
}

func TestParseRangeHeader_Empty(t *testing.T) {
	_, _, err := ParseRangeHeader("", 1000)
	if !errors.Is(err, ErrRangeNotSatisfiable) {
		t.Fatalf("expected ErrRangeNotSatisfiable, got %v", err)
	}
}

func TestParseRangeHeader_InvalidUnit(t *testing.T) {
	_, _, err := ParseRangeHeader("foo=0-1", 1000)
	if !errors.Is(err, ErrRangeNotSatisfiable) {
		t.Fatalf("expected ErrRangeNotSatisfiable, got %v", err)
	}
}

func TestParseRangeHeader_MultipleRanges(t *testing.T) {
	_, _, err := ParseRangeHeader("bytes=0-499,600-700", 1000)
	if !errors.Is(err, ErrRangeNotSatisfiable) {
		t.Fatalf("expected ErrRangeNotSatisfiable, got %v", err)
	}
}

func TestParseRangeHeader_StartPastEnd(t *testing.T) {
	_, _, err := ParseRangeHeader("bytes=500-400", 1000)
	if !errors.Is(err, ErrRangeNotSatisfiable) {
		t.Fatalf("expected ErrRangeNotSatisfiable, got %v", err)
	}
}

func TestParseRangeHeader_StartOutOfBounds(t *testing.T) {
	_, _, err := ParseRangeHeader("bytes=1000-2000", 1000)
	if !errors.Is(err, ErrRangeNotSatisfiable) {
		t.Fatalf("expected ErrRangeNotSatisfiable, got %v", err)
	}
}

func TestIterFileRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.bin")
	content := bytes.Repeat([]byte("abcdefgh"), 8192) // 64 KiB
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := IterFileRange(path, 0, int64(len(content)-1))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("expected %d bytes, got %d", len(content), len(got))
	}
}

func TestIterFileRange_Partial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.bin")
	content := []byte("0123456789")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := IterFileRange(path, 2, 7)
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if !bytes.Equal(got, []byte("234567")) {
		t.Fatalf("expected 234567, got %q", got)
	}
}

func TestIterFileRange_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.bin")

	r := IterFileRange(path, 0, 10)
	_, err := io.ReadAll(r)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestComputePeaksBuckets(t *testing.T) {
	restore := mockFfmpeg(f32leBytes(1, 2, -3, 4, -5, 6))
	defer restore()

	peaks, err := ComputePeaks("dummy.wav", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// chunk size = 6/2 = 3 -> max abs [3, 6]; normalize by 6 -> [0.5, 1]
	want := []float64{0.5, 1}
	if !sliceEqual(peaks, want) {
		t.Fatalf("expected %v, got %v", want, peaks)
	}
}

func TestComputePeaksNoNormalize(t *testing.T) {
	restore := mockFfmpeg(f32leBytes(0.5, -0.25, 0.1, -0.2))
	defer restore()

	peaks, err := ComputePeaks("dummy.wav", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []float64{0.5, 0.2}
	if !sliceEqual(peaks, want) {
		t.Fatalf("expected %v, got %v", want, peaks)
	}
}

func TestComputePeaksNormalizeSingle(t *testing.T) {
	restore := mockFfmpeg(f32leBytes(-2.5))
	defer restore()

	peaks, err := ComputePeaks("dummy.wav", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []float64{1}
	if !sliceEqual(peaks, want) {
		t.Fatalf("expected %v, got %v", want, peaks)
	}
}

func TestComputePeaksEmpty(t *testing.T) {
	restore := mockFfmpeg([]byte{})
	defer restore()

	peaks, err := ComputePeaks("dummy.wav", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peaks) != 0 {
		t.Fatalf("expected empty peaks, got %v", peaks)
	}
}

func TestComputePeaksBucketSizeZero(t *testing.T) {
	restore := mockFfmpeg(f32leBytes(1, 2, 3, -4, 0.5, 0.1))
	defer restore()

	// request more samples than available PCM samples -> bucket_size == 0
	peaks, err := ComputePeaks("dummy.wav", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peaks) != 10 {
		t.Fatalf("expected 10 peaks, got %d", len(peaks))
	}
	maxAbs := 4.0
	for i, v := range peaks {
		if math.Abs(v-maxAbs/maxAbs) > 1e-9 {
			t.Fatalf("peak %d expected normalized max %v, got %v", i, maxAbs/maxAbs, v)
		}
	}
}

func TestComputePeaksFfmpegFailure(t *testing.T) {
	restore := mockFfmpegError(errors.New("ffmpeg exploded"))
	defer restore()

	peaks, err := ComputePeaks("dummy.wav", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peaks) != 0 {
		t.Fatalf("expected empty peaks on ffmpeg failure, got %v", peaks)
	}
}

func mockFfmpeg(out []byte) func() {
	old := ffmpegRun
	ffmpegRun = func([]string) ([]byte, error) {
		return out, nil
	}
	return func() { ffmpegRun = old }
}

func mockFfmpegError(err error) func() {
	old := ffmpegRun
	ffmpegRun = func([]string) ([]byte, error) {
		return nil, err
	}
	return func() { ffmpegRun = old }
}

func sliceEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-9 {
			return false
		}
	}
	return true
}
