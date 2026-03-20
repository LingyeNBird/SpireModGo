package i18n

import (
	"encoding/json"
	"testing"
)

func TestLocalizerLoadsAndToggles(testingT *testing.T) {
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

func TestLocaleCatalogsHaveMatchingKeys(testingT *testing.T) {
	zhCatalog := mustDecodeCatalog(testingT, zhCNLocaleCatalogJSON)
	enCatalog := mustDecodeCatalog(testingT, enUSLocaleCatalogJSON)
	for key := range zhCatalog {
		if _, ok := enCatalog[key]; !ok {
			testingT.Fatalf("missing en-US key %q", key)
		}
	}
	for key := range enCatalog {
		if _, ok := zhCatalog[key]; !ok {
			testingT.Fatalf("missing zh-CN key %q", key)
		}
	}
}

func TestLocalizerBackfilledKeysAndTemplates(testingT *testing.T) {
	assertLocaleText(testingT, LocaleZhCN, "Select one or more installed mods before uninstalling.", nil, "请先选择一个或多个已安装模组再卸载。")
	assertLocaleText(testingT, LocaleEnUS, "Select one or more installed mods before uninstalling.", nil, "Select one or more installed mods before uninstalling.")
	assertLocaleText(testingT, LocaleZhCN, "Copy Options", nil, "复制选项")
	assertLocaleText(testingT, LocaleEnUS, "Copy Options", nil, "Copy Options")
	assertLocaleText(testingT, LocaleZhCN, "Select a backup before restoring.", nil, "请先选择一个备份再恢复。")
	assertLocaleText(testingT, LocaleEnUS, "Select a backup before restoring.", nil, "Select a backup before restoring.")
	assertLocaleText(testingT, LocaleZhCN, "Select a backup before deleting.", nil, "请先选择一个备份再删除。")
	assertLocaleText(testingT, LocaleEnUS, "Select a backup before deleting.", nil, "Select a backup before deleting.")
	assertLocaleText(testingT, LocaleZhCN, "Mods were installed, but no vanilla save slots were found to copy into modded mode yet.", nil, "模组已安装，但还没有找到可复制到模组存档的原版存档槽位。")
	assertLocaleText(testingT, LocaleEnUS, "Mods were installed, but no vanilla save slots were found to copy into modded mode yet.", nil, "Mods were installed, but no vanilla save slots were found to copy into modded mode yet.")
	assertLocaleText(testingT, LocaleZhCN, "Run file dialog failed: %v", []any{"boom"}, "运行文件选择对话框失败：boom")
	assertLocaleText(testingT, LocaleEnUS, "Run file dialog failed: %v", []any{"boom"}, "Run file dialog failed: boom")
	assertLocaleText(testingT, LocaleZhCN, "Run file dialog failed: %v\n\n%s", []any{"boom", "details"}, "运行文件选择对话框失败：boom\n\ndetails")
	assertLocaleText(testingT, LocaleEnUS, "Run file dialog failed: %v\n\n%s", []any{"boom", "details"}, "Run file dialog failed: boom\n\ndetails")
	assertLocaleText(testingT, LocaleZhCN, "Close", nil, "关闭")
	assertLocaleText(testingT, LocaleEnUS, "Close", nil, "Close")
	assertLocaleText(testingT, LocaleZhCN, "Confirm", nil, "确认")
	assertLocaleText(testingT, LocaleEnUS, "Confirm", nil, "Confirm")
	assertLocaleText(testingT, LocaleZhCN, "Cancel", nil, "取消")
	assertLocaleText(testingT, LocaleEnUS, "Cancel", nil, "Cancel")
	assertLocaleText(testingT, LocaleZhCN, "Copy", nil, "复制")
	assertLocaleText(testingT, LocaleEnUS, "Copy", nil, "Copy")
	assertLocaleText(testingT, LocaleZhCN, "Backup and Copy", nil, "备份并复制")
	assertLocaleText(testingT, LocaleEnUS, "Backup and Copy", nil, "Backup and Copy")
	assertLocaleText(testingT, LocaleZhCN, "Choose a destination save slot", nil, "选择目标存档槽位")
	assertLocaleText(testingT, LocaleEnUS, "Choose a destination save slot", nil, "Choose a destination save slot")
}

func mustDecodeCatalog(testingT *testing.T, raw string) map[string]string {
	testingT.Helper()
	catalog := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &catalog); err != nil {
		testingT.Fatalf("decode catalog: %v", err)
	}
	return catalog
}

func assertLocaleText(testingT *testing.T, locale, key string, args []any, want string) {
	testingT.Helper()
	if err := Load(""); err != nil {
		testingT.Fatal(err)
	}
	setLocale(testingT, locale)
	got := T(key, args...)
	if got != want {
		testingT.Fatalf("locale %s key %q: want %q, got %q", locale, key, want, got)
	}
}

func setLocale(testingT *testing.T, locale string) {
	testingT.Helper()
	if Current() == locale {
		return
	}
	if got := Toggle(); got != locale {
		testingT.Fatalf("expected locale %s, got %s", locale, got)
	}
}
