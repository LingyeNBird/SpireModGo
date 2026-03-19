package ui

import (
	"encoding/json"
	"fmt"
)

const (
	localeZhCN = "zh-CN"
	localeEnUS = "en-US"
)

type localizer struct {
	current  string
	catalogs map[string]map[string]string
}

var activeLocalizer = &localizer{current: localeZhCN, catalogs: map[string]map[string]string{}}

func loadLocalizer(_ string) error {
	activeLocalizer.catalogs = map[string]map[string]string{}
	for locale, raw := range embeddedLocaleCatalogs {
		catalog := map[string]string{}
		if err := json.Unmarshal([]byte(raw), &catalog); err != nil {
			return fmt.Errorf("load locale %s: %w", locale, err)
		}
		activeLocalizer.catalogs[locale] = catalog
	}
	activeLocalizer.current = localeZhCN
	return nil
}

func toggleLocale() string {
	if activeLocalizer.current == localeZhCN {
		activeLocalizer.current = localeEnUS
	} else {
		activeLocalizer.current = localeZhCN
	}
	return activeLocalizer.current
}

func currentLocale() string {
	return activeLocalizer.current
}

func t(key string, args ...any) string {
	text := key
	if catalog, ok := activeLocalizer.catalogs[activeLocalizer.current]; ok {
		if translated, exists := catalog[key]; exists {
			text = translated
		}
	}
	if text == key {
		if catalog, ok := activeLocalizer.catalogs[localeEnUS]; ok {
			if translated, exists := catalog[key]; exists {
				text = translated
			}
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}
