package ui

import uii18n "spiremodgo/internal/ui/i18n"

const (
	localeZhCN = uii18n.LocaleZhCN
	localeEnUS = uii18n.LocaleEnUS
)

func loadLocalizer(baseDir string) error {
	return uii18n.Load(baseDir)
}

func toggleLocale() string {
	return uii18n.Toggle()
}

func currentLocale() string {
	return uii18n.Current()
}

func t(key string, args ...any) string {
	return uii18n.T(key, args...)
}
