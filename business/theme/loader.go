// Package theme handles loading and watching the Omarchy color theme.
package theme

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

var (
	ErrThemeFileNotFound        = errors.New("theme file not found")
	ErrCouldNotLoadOmarchy4File = errors.New("could not load Omarchy 4 file")
	ErrCouldNotLoadOmarchy3File = errors.New("could not load Omarchy 3 file")
)

// ThemeColors is the version-independent color set exposed to the frontend.
type ThemeColors struct {
	Background          string `json:"background"`
	Foreground          string `json:"foreground"`
	Accent              string `json:"accent"`
	Cursor              string `json:"cursor"`
	SelectionBackground string `json:"selectionBackground"`
	SelectionForeground string `json:"selectionForeground"`
	Color0              string `json:"color0"`
	Color1              string `json:"color1"`
	Color2              string `json:"color2"`
	Color3              string `json:"color3"`
	Color4              string `json:"color4"`
	Color5              string `json:"color5"`
	Color6              string `json:"color6"`
	Color7              string `json:"color7"`
	Color8              string `json:"color8"`
	Color9              string `json:"color9"`
	Color10             string `json:"color10"`
	Color11             string `json:"color11"`
	Color12             string `json:"color12"`
	Color13             string `json:"color13"`
	Color14             string `json:"color14"`
	Color15             string `json:"color15"`
}

// Omarchy3Colors represents the colors.toml schema used by Omarchy 3.
type Omarchy3Colors struct {
	Background          string `toml:"background"`
	Foreground          string `toml:"foreground"`
	Accent              string `toml:"accent"`
	Cursor              string `toml:"cursor"`
	SelectionBackground string `toml:"selection_background"`
	SelectionForeground string `toml:"selection_foreground"`
	Color0              string `toml:"color0"`
	Color1              string `toml:"color1"`
	Color2              string `toml:"color2"`
	Color3              string `toml:"color3"`
	Color4              string `toml:"color4"`
	Color5              string `toml:"color5"`
	Color6              string `toml:"color6"`
	Color7              string `toml:"color7"`
	Color8              string `toml:"color8"`
	Color9              string `toml:"color9"`
	Color10             string `toml:"color10"`
	Color11             string `toml:"color11"`
	Color12             string `toml:"color12"`
	Color13             string `toml:"color13"`
	Color14             string `toml:"color14"`
	Color15             string `toml:"color15"`
}

// Omarchy4Colors represents the semantic colors.toml schema used by Omarchy 4.
type Omarchy4Colors struct {
	Mode              string `toml:"mode"`
	Accent            string `toml:"accent"`
	Selection         string `toml:"selection"`
	Muted             string `toml:"muted"`
	Background        string `toml:"background"`
	DarkBackground    string `toml:"dark_background"`
	DarkerBackground  string `toml:"darker_background"`
	LighterBackground string `toml:"lighter_background"`
	Foreground        string `toml:"foreground"`
	DarkForeground    string `toml:"dark_foreground"`
	LightForeground   string `toml:"light_foreground"`
	BrightForeground  string `toml:"bright_foreground"`
	Red               string `toml:"red"`
	Yellow            string `toml:"yellow"`
	Orange            string `toml:"orange"`
	Green             string `toml:"green"`
	Cyan              string `toml:"cyan"`
	Blue              string `toml:"blue"`
	Magenta           string `toml:"magenta"`
	Brown             string `toml:"brown"`
	BrightRed         string `toml:"bright_red"`
	BrightYellow      string `toml:"bright_yellow"`
	BrightGreen       string `toml:"bright_green"`
	BrightCyan        string `toml:"bright_cyan"`
	BrightBlue        string `toml:"bright_blue"`
	BrightMagenta     string `toml:"bright_magenta"`
}

// Load reads either supported Omarchy colors.toml schema and returns the common frontend representation.
func Load(path string) (ThemeColors, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ThemeColors{}, fmt.Errorf("%w: %w", ErrThemeFileNotFound, err)
	}

	if areWeInOmarchy4CompatibleColorsFile(data) {
		var colors Omarchy4Colors
		if err := toml.Unmarshal(data, &colors); err != nil {
			return ThemeColors{}, fmt.Errorf("%w: %w", ErrCouldNotLoadOmarchy4File, err)
		}
		return colors.ThemeColors(), nil
	}

	var colors Omarchy3Colors
	if err := toml.Unmarshal(data, &colors); err != nil {
		return ThemeColors{}, fmt.Errorf("%w: %w", ErrCouldNotLoadOmarchy3File, err)
	}
	return colors.ThemeColors(), nil
}

func areWeInOmarchy4CompatibleColorsFile(data []byte) bool {
	var schema struct {
		Mode string `toml:"mode"`
	}
	if err := toml.Unmarshal(data, &schema); err != nil {
		return false
	}
	return schema.Mode != ""
}

// ThemeColors converts Omarchy 3 colors to the common frontend representation.
func (c Omarchy3Colors) ThemeColors() ThemeColors {
	return ThemeColors(c)
}

// ThemeColors maps Omarchy 4 semantic colors onto the common frontend representation.
func (c Omarchy4Colors) ThemeColors() ThemeColors {
	return ThemeColors{
		Background: c.Background, Foreground: c.Foreground, Accent: c.Accent,
		Cursor: c.BrightForeground, SelectionBackground: c.Selection, SelectionForeground: c.BrightForeground,
		Color0: c.LighterBackground, Color1: c.Red, Color2: c.Green, Color3: c.Yellow,
		Color4: c.Blue, Color5: c.Magenta, Color6: c.Cyan, Color7: c.Foreground,
		Color8: c.Muted, Color9: c.BrightRed, Color10: c.BrightGreen, Color11: c.BrightYellow,
		Color12: c.BrightBlue, Color13: c.BrightMagenta, Color14: c.BrightCyan, Color15: c.BrightForeground,
	}
}
