package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalizerLoadsAndToggles(testingT *testing.T) {
	testingT.Parallel()
	baseDir := testingT.TempDir()
	langDir := filepath.Join(baseDir, "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		testingT.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "zh-CN.json"), []byte(`{"Hello":"你好","Value %d":"值 %d"}`), 0o644); err != nil {
		testingT.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "en-US.json"), []byte(`{"Hello":"Hello","Value %d":"Value %d"}`), 0o644); err != nil {
		testingT.Fatal(err)
	}
	if err := loadLocalizer(baseDir); err != nil {
		testingT.Fatal(err)
	}
	if currentLocale() != localeZhCN {
		testingT.Fatalf("expected default locale %s, got %s", localeZhCN, currentLocale())
	}
	if got := t("Hello"); got != "你好" {
		testingT.Fatalf("expected zh translation, got %q", got)
	}
	if got := t("Value %d", 3); got != "值 3" {
		testingT.Fatalf("expected formatted zh translation, got %q", got)
	}
	if got := toggleLocale(); got != localeEnUS {
		testingT.Fatalf("expected toggled locale %s, got %s", localeEnUS, got)
	}
	if got := t("Hello"); got != "Hello" {
		testingT.Fatalf("expected en translation, got %q", got)
	}
	if got := t("Missing Key"); got != "Missing Key" {
		testingT.Fatalf("expected missing key fallback, got %q", got)
	}
}
