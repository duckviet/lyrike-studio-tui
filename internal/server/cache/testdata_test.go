package cache

// Sample payloads used by the Store tests. Pulled into a separate file so
// the test logic in store_test.go stays under the per-file review ceiling.

func sampleMetadata() map[string]any {
	return map[string]any{
		"id":        "dQw4w9WgXcQ",
		"title":     "Never Gonna Give You Up",
		"uploader":  "Rick Astley",
		"duration":  213.0,
		"fetchedAt": "2025-06-22T12:00:00Z",
	}
}

func samplePeaks() map[string]any {
	return map[string]any{
		"version":  1,
		"source":   "original",
		"duration": 213.0,
		"samples":  []any{0.0, 0.25, 0.5, 0.75, 1.0},
	}
}

func sampleTranscript() map[string]any {
	return map[string]any{
		"version":  1,
		"language": "en",
		"lines": []any{
			map[string]any{"t": 0.0, "d": 3.5, "text": "We're no strangers to love"},
			map[string]any{"t": 3.5, "d": 4.0, "text": "You know the rules and so do I"},
		},
	}
}
