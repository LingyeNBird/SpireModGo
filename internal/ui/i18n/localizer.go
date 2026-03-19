package i18n

import (
	"encoding/json"
	"fmt"
)

const (
	LocaleZhCN = "zh-CN"
	LocaleEnUS = "en-US"
)

type localizer struct {
	current  string
	catalogs map[string]map[string]string
}

var activeLocalizer = &localizer{current: LocaleZhCN, catalogs: map[string]map[string]string{}}

func Load(_ string) error {
	activeLocalizer.catalogs = map[string]map[string]string{}
	for locale, raw := range embeddedLocaleCatalogs {
		catalog := map[string]string{}
		if err := json.Unmarshal([]byte(raw), &catalog); err != nil {
			return fmt.Errorf("load locale %s: %w", locale, err)
		}
		activeLocalizer.catalogs[locale] = catalog
	}
	activeLocalizer.current = LocaleZhCN
	return nil
}

func Toggle() string {
	if activeLocalizer.current == LocaleZhCN {
		activeLocalizer.current = LocaleEnUS
	} else {
		activeLocalizer.current = LocaleZhCN
	}
	return activeLocalizer.current
}

func Current() string {
	return activeLocalizer.current
}

func T(key string, args ...any) string {
	text := key
	if catalog, ok := activeLocalizer.catalogs[activeLocalizer.current]; ok {
		if translated, exists := catalog[key]; exists {
			text = translated
		}
	}
	if text == key {
		if catalog, ok := activeLocalizer.catalogs[LocaleEnUS]; ok {
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
