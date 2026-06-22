package tui

import (
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
)

func ptr(s string) *string { return &s }

func TestFetchMediaMsgSetsProjectIDWhenEmpty(t *testing.T) {
	url := "https://example.com/source"
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, &memoryDraftStore{}, "", "", "")

	next, _ := model.Update(fetchMediaMsg{resp: backend.FetchResponse{VideoID: "P0N0h_EOS-c", SourceURL: ptr(url)}})
	got := next.(Model)

	if got.projectID != draft.ProjectID("P0N0h_EOS-c") {
		t.Fatalf("projectID = %q, want P0N0h_EOS-c", got.projectID)
	}
	if got.sourceURL != url {
		t.Fatalf("sourceURL = %q, want %q", got.sourceURL, url)
	}
}

func TestFetchMediaMsgPreservesNonEmptyProjectID(t *testing.T) {
	url := "https://example.com/source"
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, &memoryDraftStore{}, draft.ProjectID("existing"), "", "")

	next, _ := model.Update(fetchMediaMsg{resp: backend.FetchResponse{VideoID: "P0N0h_EOS-c", SourceURL: ptr(url)}})
	got := next.(Model)

	if got.projectID != draft.ProjectID("existing") {
		t.Fatalf("projectID = %q, want existing", got.projectID)
	}
	if got.sourceURL != url {
		t.Fatalf("sourceURL = %q, want %q", got.sourceURL, url)
	}
}
