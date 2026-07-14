package i18n

import (
	"strings"
	"testing"
)

func TestNormalizeAndNext(t *testing.T) {
	if Normalize("pt-br") != PT || Normalize("en") != EN {
		t.Fatal("Normalize should accept the two supported languages")
	}
	if Normalize("") != EN || Normalize("xx") != EN {
		t.Fatal("Normalize should default unknown/empty to EN")
	}
	if Next(EN) != PT || Next(PT) != EN {
		t.Fatal("Next should flip between the two languages")
	}
}

// TestCatalogComplete guards against half-translated entries: every key must
// have both an English source and a pt-br translation, and the English source
// must never be empty (it is the fallback for missing translations).
func TestCatalogComplete(t *testing.T) {
	for key, pair := range catalog {
		if strings.TrimSpace(pair[0]) == "" {
			t.Errorf("key %q has an empty English (canonical) value", key)
		}
		if strings.TrimSpace(pair[1]) == "" {
			t.Errorf("key %q has an empty pt-br translation", key)
		}
	}
}

func TestLookupFallback(t *testing.T) {
	if got := EN.T("banner.subtitle"); got != "tasks in your terminal" {
		t.Errorf("unexpected en lookup: %q", got)
	}
	if got := PT.T("banner.subtitle"); got != "tarefas no terminal" {
		t.Errorf("unexpected pt lookup: %q", got)
	}
	if got := EN.T("does.not.exist"); got != "does.not.exist" {
		t.Errorf("missing key should return the key itself, got %q", got)
	}
	if got := PT.Tf("status.taskDone", 7); got != "  ✓ tarefa 7 concluída" {
		t.Errorf("Tf formatting failed: %q", got)
	}
}
