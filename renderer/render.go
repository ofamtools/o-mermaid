package renderer

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/chromedp/chromedp"
)

//go:embed beautiful-mermaid.js
var engine string

//go:embed themes.json
var themeData []byte

func Themes() map[string]map[string]string {
	var themes map[string]map[string]string
	if err := json.Unmarshal(themeData, &themes); err != nil {
		panic(err)
	}
	return themes
}

type Options struct {
	Theme      string
	Format     string
	Padding    int
	Background string
	Chrome     string
}

func browser(ctx context.Context, executable string) (context.Context, func()) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if executable != "" {
		opts = append(opts, chromedp.ExecPath(executable))
	}
	alloc, stop := chromedp.NewExecAllocator(ctx, opts...)
	tab, closeTab := chromedp.NewContext(alloc)
	return tab, func() { closeTab(); stop() }
}

func Render(ctx context.Context, code string, o Options) ([]byte, error) {
	theme, ok := Themes()[o.Theme]
	if !ok {
		return nil, fmt.Errorf("unknown theme %q; run themes", o.Theme)
	}
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("Mermaid input is empty")
	}
	tab, closeTab := browser(ctx, o.Chrome)
	defer closeTab()
	source, _ := json.Marshal(code)
	colors, _ := json.Marshal(theme)
	if err := chromedp.Do(tab,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate[chromedp.Void](engine),
	); err != nil {
		return nil, fmt.Errorf("start renderer (Chrome/Chromium required): %w", err)
	}
	svg, err := chromedp.Run(tab, chromedp.Evaluate[string](fmt.Sprintf("beautifulMermaid.renderMermaidSVG(%s, %s)", source, colors)))
	if err != nil {
		return nil, fmt.Errorf("render Mermaid: %w", err)
	}
	switch o.Format {
	case "svg":
		return []byte(svg), nil
	case "html":
		return HTML(svg, "Mermaid diagram", o.Padding, o.Background)
	case "png":
		document, err := HTML(svg, "Mermaid diagram", o.Padding, o.Background)
		if err != nil {
			return nil, err
		}
		err = chromedp.Do(tab,
			chromedp.EmulateViewport(3840, 2160),
			chromedp.Navigate("data:text/html;base64,"+base64.StdEncoding.EncodeToString(document)),
			chromedp.Evaluate[chromedp.Void](`document.querySelector("img").decode()`, chromedp.EvalAwaitPromise),
		)
		if err != nil {
			return nil, fmt.Errorf("capture PNG: %w", err)
		}
		return chromedp.Run(tab, chromedp.FullScreenshot(100))
	default:
		return nil, fmt.Errorf("unsupported format %q", o.Format)
	}
}

var backgroundPattern = regexp.MustCompile(`(?i)(?:background(?:-color)?|--bg):\s*([^;"\s]+)`)
var rectPattern = regexp.MustCompile(`<rect[^>]*fill="([^"]+)"[^>]*(?:width="100%"|height="100%")`)
var colorPattern = regexp.MustCompile(`^(?:#[0-9a-fA-F]{3,8}|[a-zA-Z]+|(?:rgb|rgba|hsl|hsla)\([0-9.,%\s+/-]+\))$`)

func HTML(svg, title string, padding int, background string) ([]byte, error) {
	if padding < 0 {
		return nil, fmt.Errorf("padding must not be negative")
	}
	if background == "" {
		for _, pattern := range []*regexp.Regexp{backgroundPattern, rectPattern} {
			if match := pattern.FindStringSubmatch(svg); len(match) > 1 && colorPattern.MatchString(match[1]) {
				background = match[1]
				break
			}
		}
		if background == "" {
			background = "#ffffff"
		}
	}
	if !colorPattern.MatchString(background) {
		return nil, fmt.Errorf("unsupported background color %q", background)
	}
	// SVG remains an image, not executable markup in the wrapper document.
	image := base64.StdEncoding.EncodeToString([]byte(svg))
	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s</title>
<style>*{margin:0;padding:0;box-sizing:border-box}html,body{background:%s}.container{display:inline-block;padding:%dpx}.container img{display:block;min-width:1200px;height:auto}</style>
</head><body><div class="container"><img alt="Mermaid diagram" src="data:image/svg+xml;base64,%s"></div></body></html>
`, html.EscapeString(title), background, padding, image)), nil
}
