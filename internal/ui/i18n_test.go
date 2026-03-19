package ui

import "testing"

func TestLocalizerLoadsAndToggles(testingT *testing.T) {
	testingT.Parallel()
	if err := loadLocalizer(""); err != nil {
		testingT.Fatal(err)
	}
	if currentLocale() != localeZhCN {
		testingT.Fatalf("expected default locale %s, got %s", localeZhCN, currentLocale())
	}
	if got := t("Settings"); got != "设置" {
		testingT.Fatalf("expected zh translation, got %q", got)
	}
	if got := t("Installed [%d]", 3); got != "已安装[3]" {
		testingT.Fatalf("expected formatted zh translation, got %q", got)
	}
	if got := toggleLocale(); got != localeEnUS {
		testingT.Fatalf("expected toggled locale %s, got %s", localeEnUS, got)
	}
	if got := t("Settings"); got != "Settings" {
		testingT.Fatalf("expected en translation, got %q", got)
	}
	if got := t("Missing Key"); got != "Missing Key" {
		testingT.Fatalf("expected missing key fallback, got %q", got)
	}
}
