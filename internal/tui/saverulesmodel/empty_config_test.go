package saverulesmodel

import (
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/helpers"
	"strings"
	"testing"
)

// Deleting every repository is a supported state. The save flow has nothing to
// write to then, and must say so clearly instead of panicking or telling the
// user to "run setup first" — a setup wizard they can no longer reach.

func TestNewSaveRulesModel_NoRepositoriesConfigured(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	cfg := &config.Config{Repositories: []repository.RepositoryEntry{}}
	ctx := helpers.NewUIContext(80, 24, cfg, logger)

	m := NewSaveRulesModel(ctx)

	if m.state != StateError {
		t.Fatalf("expected StateError with no repositories, got %v", m.state)
	}
	if m.err == nil {
		t.Fatal("expected an error explaining there is nowhere to save")
	}
	if !strings.Contains(m.err.Error(), "no repositories configured") {
		t.Errorf("expected the empty-config message, got %q", m.err.Error())
	}
	if !strings.Contains(m.err.Error(), "Add New Repository") {
		t.Errorf("expected the error to point at Add New Repository, got %q", m.err.Error())
	}

	// Init must preserve the constructor's error instead of replacing it with
	// the generic "FileManager not initialized" message.
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init to do nothing while in the error state")
	}

	// Rendering the error screen must not panic, and must show the real reason.
	view := m.View()
	if view == "" {
		t.Fatal("expected the error view to render")
	}
	if strings.Contains(view, "FileManager not initialized") {
		t.Errorf("empty config should not surface the generic FileManager error, got:\n%s", view)
	}
	if !strings.Contains(view, "no repositories configured") {
		t.Errorf("expected the empty-config explanation in the view, got:\n%s", view)
	}
}

func TestNoUsableRepositoriesError_DistinguishesUnavailable(t *testing.T) {
	empty := noUsableRepositoriesError(0, "save rules to").Error()
	if !strings.Contains(empty, "no repositories configured") {
		t.Errorf("expected empty-config wording, got %q", empty)
	}

	broken := noUsableRepositoriesError(2, "save rules to").Error()
	if strings.Contains(broken, "no repositories configured") {
		t.Errorf("configured-but-unavailable should not read as unconfigured, got %q", broken)
	}
	if !strings.Contains(broken, "2 configured repositories") {
		t.Errorf("expected the configured count, got %q", broken)
	}

	if single := noUsableRepositoriesError(1, "save rules to").Error(); !strings.Contains(single, "1 configured repository") {
		t.Errorf("expected singular wording for one repository, got %q", single)
	}
}
