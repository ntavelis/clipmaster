package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOmarchy3Colors(t *testing.T) {
	path := writeThemeFile(t, `
background = "bg"
foreground = "fg"
accent = "accent"
cursor = "cursor"
selection_background = "selection-bg"
selection_foreground = "selection-fg"
color0 = "c0"
color15 = "c15"
`)

	colors, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if colors.Background != "bg" || colors.SelectionBackground != "selection-bg" || colors.Color0 != "c0" || colors.Color15 != "c15" {
		t.Fatalf("unexpected Omarchy 3 conversion: %+v", colors)
	}
}

func TestLoadOmarchy4Colors(t *testing.T) {
	path := writeThemeFile(t, `
mode = "dark"
accent = "accent"
selection = "selection"
muted = "muted"
background = "background"
lighter_background = "lighter-background"
foreground = "foreground"
dark_foreground = "dark-foreground"
bright_foreground = "bright-foreground"
red = "red"
yellow = "yellow"
green = "green"
cyan = "cyan"
blue = "blue"
magenta = "magenta"
bright_red = "bright-red"
bright_yellow = "bright-yellow"
bright_green = "bright-green"
bright_cyan = "bright-cyan"
bright_blue = "bright-blue"
bright_magenta = "bright-magenta"
`)

	colors, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if colors.Cursor != "bright-foreground" || colors.SelectionBackground != "selection" || colors.Color0 != "background" || colors.Color7 != "foreground" {
		t.Fatalf("unexpected Omarchy 4 base conversion: %+v", colors)
	}
	if colors.Color1 != "red" || colors.Color10 != "bright-green" || colors.Color15 != "bright-foreground" {
		t.Fatalf("unexpected Omarchy 4 ANSI conversion: %+v", colors)
	}
}

func writeThemeFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "colors.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
