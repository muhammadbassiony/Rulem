package importrulesmenu

import (
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/helpers"
	"strings"
	"testing"
)

// Deleting every repository is a supported state. The import flow has nothing to
// read from then, and must say so clearly instead of panicking or telling the
// user to "run setup first" — a setup wizard they can no longer reach.

func TestNewImportRulesModel_NoRepositoriesConfigured(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	cfg := &config.Config{Repositories: []repository.RepositoryEntry{}}
	ctx := helpers.NewUIContext(80, 24, cfg, logger)

	m := NewImportRulesModel(ctx)

	if m.state != StateError {
		t.Fatalf("expected StateError with no repositories, got %v", m.state)
	}
	if m.err == nil {
		t.Fatal("expected an error explaining there is nothing to import")
	}
	if !strings.Contains(m.err.Error(), "no repositories configured") {
		t.Errorf("expected the empty-config message, got %q", m.err.Error())
	}
	if !strings.Contains(m.err.Error(), "Add New Repository") {
		t.Errorf("expected the error to point at Add New Repository, got %q", m.err.Error())
	}

	// Init must preserve the error state rather than starting a scan.
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init to do nothing while in the error state")
	}

	if view := m.View(); view == "" {
		t.Error("expected the error view to render")
	}
}

func TestNoUsableRepositoriesError_DistinguishesUnavailable(t *testing.T) {
	empty := noUsableRepositoriesError(0).Error()
	if !strings.Contains(empty, "no repositories configured") {
		t.Errorf("expected empty-config wording, got %q", empty)
	}

	broken := noUsableRepositoriesError(3).Error()
	if strings.Contains(broken, "no repositories configured") {
		t.Errorf("configured-but-unavailable should not read as unconfigured, got %q", broken)
	}
	if !strings.Contains(broken, "3 configured repositories") {
		t.Errorf("expected the configured count, got %q", broken)
	}
}
