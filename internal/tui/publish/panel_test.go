package publish

import (
	"errors"
	"strings"
	"testing"
)

func TestPublishFlowValidatePowPublishDone(t *testing.T) {
	t.Parallel()

	panel := NewPanel()
	panel, err := panel.Validate("[00:01.00]Line")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	panel, err = panel.SolveChallenge("prefix-", "nonce")
	if err != nil {
		t.Fatalf("SolveChallenge() error = %v", err)
	}
	panel = panel.Publish(nil)

	if panel.State() != StateDone {
		t.Fatalf("State() = %q, want %q", panel.State(), StateDone)
	}
	if panel.Token() != "prefix-nonce" {
		t.Fatalf("Token() = %q, want prefix-nonce", panel.Token())
	}
}

func TestPublishFailureRetryReturnsToPow(t *testing.T) {
	t.Parallel()

	panel := NewPanel()
	panel, err := panel.Validate("[00:01.00]Line")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	panel, err = panel.SolveChallenge("prefix-", "nonce")
	if err != nil {
		t.Fatalf("SolveChallenge() error = %v", err)
	}

	panel = panel.Publish(errors.New("upstream rejected"))
	panel = panel.Retry()

	if panel.State() != StatePoW {
		t.Fatalf("State() = %q, want %q", panel.State(), StatePoW)
	}
	if panel.RetryCount() != 1 {
		t.Fatalf("RetryCount() = %d, want 1", panel.RetryCount())
	}
	if !strings.Contains(strings.ToLower(panel.View(80, 1)), "proof-of-work") {
		t.Fatalf("View() = %q, want state %q", panel.View(80, 1), "proof-of-work")
	}
}
