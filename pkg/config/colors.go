package config

import "github.com/fatih/color"

type colorDef struct {
	Fg        string
	Bg        string
	Bold      bool
	Faint     bool
	Italic    bool
	Underline bool
}

type ColorConfig struct {
	Timestamp      colorDef
	AttributeKey   colorDef
	AttributeValue colorDef
	Message        colorDef
}

var defaultColorConfig ColorConfig = ColorConfig{
	Timestamp: colorDef{
		Fg: "magenta",
	},
	Message: colorDef{
		Bold: true,
	},
	AttributeKey: colorDef{
		Fg: "cyan",
	},
}

func colorStringToAttribute(s string, fg bool) color.Attribute {
	fgOrBg := func(f color.Attribute, b color.Attribute) color.Attribute {
		if fg {
			return f
		} else {
			return b
		}
	}
	switch s {
	case "red":
		return fgOrBg(color.FgRed, color.BgRed)
	case "green":
		return fgOrBg(color.FgGreen, color.BgGreen)
	case "yellow":
		return fgOrBg(color.FgYellow, color.BgYellow)
	case "blue":
		return fgOrBg(color.FgBlue, color.BgBlue)
	case "magenta":
		return fgOrBg(color.FgMagenta, color.BgMagenta)
	case "cyan":
		return fgOrBg(color.FgCyan, color.BgCyan)
	case "white":
		return fgOrBg(color.FgWhite, color.BgWhite)
	}
	return color.Reset
}

func ColorToTermColor(colorDef colorDef) *color.Color {
	c := color.New()

	if colorDef.Fg != "" {
		c = c.Add(colorStringToAttribute(colorDef.Fg, true))
	}
	if colorDef.Bg != "" {
		c = c.Add(colorStringToAttribute(colorDef.Bg, false))
	}

	if colorDef.Bold {
		c = c.Add(color.Bold)
	}
	if colorDef.Faint {
		c = c.Add(color.Faint)
	}
	if colorDef.Italic {
		c = c.Add(color.Italic)
	}
	if colorDef.Underline {
		c = c.Add(color.Underline)
	}
	return c
}
