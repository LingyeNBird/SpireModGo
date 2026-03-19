package i18n

import "testing"

func TestLocalizerLoadsAndToggles(testingT *testing.T) {
	testingT.Parallel()
	if err := Load(""); err != nil {
		testingT.Fatal(err)
	}
	if Current() != LocaleZhCN {
		testingT.Fatalf("expected default locale %s, got %s", LocaleZhCN, Current())
	}
	if got := T("Settings"); got != "设置" {
		testingT.Fatalf("expected zh translation, got %q", got)
	}
	if got := T("Installed [%d]", 3); got != "已安装[3]" {
		testingT.Fatalf("expected formatted zh translation, got %q", got)
	}
	if got := Toggle(); got != LocaleEnUS {
		testingT.Fatalf("expected toggled locale %s, got %s", LocaleEnUS, got)
	}
	if got := T("Settings"); got != "Settings" {
		testingT.Fatalf("expected en translation, got %q", got)
	}
	if got := T("Missing Key"); got != "Missing Key" {
		testingT.Fatalf("expected missing key fallback, got %q", got)
	}
}
