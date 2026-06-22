package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// newTestStore returns a Store rooted at a fresh t.TempDir() and the resolved
// cache directory. Tests must use this helper to keep all I/O inside the
// per-test temp tree — never write to a real cache path.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir), dir
}

func TestStore_RoundTrip_Metadata(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)
	want := sampleMetadata()

	if err := store.SaveMetadata("dQw4w9WgXcQ", want); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	got, err := store.LoadMetadata("dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}

	if got["title"] != want["title"] {
		t.Fatalf("title = %v, want %v", got["title"], want["title"])
	}
	if got["uploader"] != want["uploader"] {
		t.Fatalf("uploader = %v, want %v", got["uploader"], want["uploader"])
	}
	if got["duration"] != want["duration"] {
		t.Fatalf("duration = %v, want %v", got["duration"], want["duration"])
	}

	// File lives exactly where the path helper promises.
	expectedPath := filepath.Join(dir, "media", "dQw4w9WgXcQ.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected metadata file at %s: %v", expectedPath, err)
	}
}

func TestStore_RoundTrip_Peaks(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)
	want := samplePeaks()

	if err := store.SavePeaks("vid-1", "original", want); err != nil {
		t.Fatalf("SavePeaks() error = %v, want nil", err)
	}

	got, err := store.LoadPeaks("vid-1", "original")
	if err != nil {
		t.Fatalf("LoadPeaks() error = %v, want nil", err)
	}

	if got["source"] != "original" {
		t.Fatalf("source = %v, want %q", got["source"], "original")
	}
	samples, ok := got["samples"].([]any)
	if !ok {
		t.Fatalf("samples type = %T, want []any", got["samples"])
	}
	if len(samples) != 5 {
		t.Fatalf("len(samples) = %d, want 5", len(samples))
	}

	// Demucs source coexists with original — different files, same store.
	if err := store.SavePeaks("vid-1", "demucs", samplePeaks()); err != nil {
		t.Fatalf("SavePeaks(demucs) error = %v", err)
	}
	original, err := store.LoadPeaks("vid-1", "original")
	if err != nil {
		t.Fatalf("LoadPeaks(original) after demucs save: error = %v", err)
	}
	if original["source"] != "original" {
		t.Fatalf("original.source = %v, want %q (demucs must not clobber original)", original["source"], "original")
	}

	// File path matches PeaksPath helper.
	expectedPath := filepath.Join(dir, "peaks", "vid-1", "original.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected peaks file at %s: %v", expectedPath, err)
	}
}

func TestStore_RoundTrip_Transcript(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)
	want := sampleTranscript()

	if err := store.SaveTranscript("vid-1", want); err != nil {
		t.Fatalf("SaveTranscript() error = %v, want nil", err)
	}

	got, err := store.LoadTranscript("vid-1")
	if err != nil {
		t.Fatalf("LoadTranscript() error = %v, want nil", err)
	}

	if got["version"] != float64(1) {
		t.Fatalf("version = %v (%T), want float64(1)", got["version"], got["version"])
	}
	if got["language"] != want["language"] {
		t.Fatalf("language = %v, want %v", got["language"], want["language"])
	}
	lines, ok := got["lines"].([]any)
	if !ok {
		t.Fatalf("lines type = %T, want []any", got["lines"])
	}
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}

	expectedPath := filepath.Join(dir, "transcripts", "vid-1.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected transcript file at %s: %v", expectedPath, err)
	}
}

func TestStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)

	_, err := store.LoadMetadata("missing")
	if err == nil {
		t.Fatalf("LoadMetadata(missing) error = nil, want ErrNotFound")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadMetadata(missing) error = %v, want ErrNotFound", err)
	}

	_, err = store.LoadPeaks("missing", "original")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadPeaks(missing) error = %v, want ErrNotFound", err)
	}

	_, err = store.LoadTranscript("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadTranscript(missing) error = %v, want ErrNotFound", err)
	}
}

func TestStore_Load_Corrupt(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)

	// Drop garbage at each path and assert the typed error.
	corrupt := []struct {
		name    string
		relPath string
		probe   func() error
	}{
		{"metadata", filepath.Join("media", "bad.json"),
			func() error { _, err := store.LoadMetadata("bad"); return err }},
		{"peaks", filepath.Join("peaks", "bad", "original.json"),
			func() error { _, err := store.LoadPeaks("bad", "original"); return err }},
		{"transcript", filepath.Join("transcripts", "bad.json"),
			func() error { _, err := store.LoadTranscript("bad"); return err }},
	}

	for _, c := range corrupt {
		full := filepath.Join(dir, c.relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", c.name, err)
		}
		if err := os.WriteFile(full, []byte("not json at all {"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", c.name, err)
		}

		err := c.probe()
		if err == nil {
			t.Fatalf("%s: probe error = nil, want ErrCorrupt", c.name)
		}
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("%s: probe error = %v, want ErrCorrupt", c.name, err)
		}
	}
}

func TestStore_Save_AutoCreatesParentDirs(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)

	// None of the parent dirs exist yet — Save must create them.
	if err := store.SaveMetadata("fresh", sampleMetadata()); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}
	if err := store.SavePeaks("fresh", "original", samplePeaks()); err != nil {
		t.Fatalf("SavePeaks() error = %v, want nil", err)
	}
	if err := store.SaveTranscript("fresh", sampleTranscript()); err != nil {
		t.Fatalf("SaveTranscript() error = %v, want nil", err)
	}

	for _, rel := range []string{
		filepath.Join("media", "fresh.json"),
		filepath.Join("peaks", "fresh", "original.json"),
		filepath.Join("transcripts", "fresh.json"),
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestStore_Save_NoLeftoverTempFiles(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)

	if err := store.SaveMetadata("clean", sampleMetadata()); err != nil {
		t.Fatalf("SaveMetadata() error = %v", err)
	}

	// No .tmp-*.json files should remain in the media dir.
	entries, err := os.ReadDir(filepath.Join(dir, "media"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || (len(e.Name()) > 4 && e.Name()[:4] == ".tmp") {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
	}
}

func TestStore_PathHelpers(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"MetadataPath", store.MetadataPath("vidA"),
			filepath.Join(dir, "media", "vidA.json")},
		{"PeaksPath", store.PeaksPath("vidA", "original"),
			filepath.Join(dir, "peaks", "vidA", "original.json")},
		{"TranscriptPath", store.TranscriptPath("vidA"),
			filepath.Join(dir, "transcripts", "vidA.json")},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestStore_RejectsPathTraversal(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)

	// A video id containing path separators must be rejected up-front, not
	// silently written outside the cache root.
	bad := []string{
		"../escape",
		"foo/bar",
		"foo\\bar",
		"..",
		"/abs",
	}
	for _, id := range bad {
		if err := store.SaveMetadata(id, sampleMetadata()); err == nil {
			t.Errorf("SaveMetadata(%q) error = nil, want validation error", id)
		}
		if err := store.SavePeaks(id, "original", samplePeaks()); err == nil {
			t.Errorf("SavePeaks(%q) error = nil, want validation error", id)
		}
		if err := store.SaveTranscript(id, sampleTranscript()); err == nil {
			t.Errorf("SaveTranscript(%q) error = nil, want validation error", id)
		}
	}
}
