package main

import (
	"errors"
	"flag"
	"github.com/jterrazz/studio-station/src/cmd/generate/pages"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/src/catalog"
)

const (
	appsStart   = "<!-- GENERATED:apps by `make generate` from src/catalog: DO NOT EDIT between markers -->"
	appsEnd     = "<!-- /GENERATED:apps -->"
	preferStart = "<!-- GENERATED:prefer by `make generate` from src/catalog: DO NOT EDIT between markers -->"
	preferEnd   = "<!-- /GENERATED:prefer -->"

	defaultPath = "skills/studio-toolbelt/SKILL.md"
)

// hazard: the boundary test keeps the process exit and the standard error stream
// to src/cmd/station, so a failure here panics: go run then exits non-zero with the
// error on stderr.
func main() {
	path := flag.String("file", defaultPath, "the skill page whose generated blocks are rewritten")
	page := flag.String("footprint", footprintPath, "the footprint page, rewritten whole")
	fixtures := flag.String("pages", pagesDir, "the folder the page fixtures are written to")
	stream := flag.String("stream", streamPath, "the recorded menubar stream, rewritten whole")
	catalogue := flag.String("calls", callsPath, "calls.json, the registry's catalogue, rewritten whole")
	stories := flag.String("stories", storiesDir, "the folder the story worlds' pages are written to")
	world := flag.String("world", pages.PlayWorldPath, "the stories' machines as an engine world, the play rig's, rewritten whole")
	schema := flag.String("schema", schemaPath, "the JSON Schema of an item file of the catalog, rewritten whole")
	flag.Parse()
	current, err := os.ReadFile(*path)
	if err != nil {
		panic(err)
	}
	rendered, err := render(string(current))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*path, []byte(rendered), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*page, []byte(footprint()), 0o600); err != nil {
		panic(err)
	}
	for name, body := range pageFixtures() {
		if err := os.WriteFile(filepath.Join(*fixtures, name), body, 0o600); err != nil {
			panic(err)
		}
	}
	if err := rewrite(*stories, storyPages()); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(*stream), 0o750); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*stream, menubarStream(), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(*stream), filepath.Base(eventsPath)), pages.EventsStream(), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*catalogue, calls(), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*world, playWorld(), 0o600); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(*schema), 0o750); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*schema, catalogSchema(), 0o600); err != nil {
		panic(err)
	}
}

func playWorld() []byte {
	base, err := os.ReadFile(pages.PlayWorldBase)
	if err != nil {
		panic(err)
	}
	return pages.PlayWorld(base)
}

func rewrite(dir string, files map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func render(source string) (string, error) {
	withApps, err := splice(source, appsStart, appsEnd, appsBlock())
	if err != nil {
		return "", err
	}
	return splice(withApps, preferStart, preferEnd, preferBlock())
}

func splice(source, start, end, block string) (string, error) {
	from := strings.Index(source, start)
	if from == -1 {
		return "", errors.New("marker not found: " + start)
	}
	afterStart := from + len(start)
	to := strings.Index(source[afterStart:], end)
	if to == -1 {
		return "", errors.New("marker not found: " + end)
	}
	return source[:afterStart] + "\n\n" + block + "\n\n" + source[afterStart+to:], nil
}

func appsBlock() string {
	var sections []string
	for _, category := range catalog.Categories {
		lines := []string{"### " + string(category), ""}
		for _, t := range builtin().Apps {
			if t.Category != category {
				continue
			}
			line := "- **" + t.Name + "**"
			if t.Description != "" {
				line += ": " + t.Description
			}
			lines = append(lines, line)
		}
		if len(lines) > 2 {
			sections = append(sections, strings.Join(lines, "\n"))
		}
	}
	return strings.Join(sections, "\n\n")
}

// invariant: columns are padded to their widest cell, the shape the formatter
// writes, so the formatter and the sync test agree on every byte.
func preferBlock() string {
	rows := [][2]string{{"Instead of", "Use"}}
	for _, t := range builtin().Apps {
		if t.Replaces != "" {
			rows = append(rows, [2]string{t.Replaces, "`" + t.Name + "`: " + t.Description})
		}
	}
	var widths [2]int
	for _, row := range rows {
		for column, cell := range row {
			widths[column] = max(widths[column], utf8.RuneCountInString(cell))
		}
	}
	lines := make([]string, 0, len(rows)+1)
	for i, row := range rows {
		lines = append(lines, "| "+pad(row[0], widths[0])+" | "+pad(row[1], widths[1])+" |")
		if i == 0 {
			lines = append(lines, "| "+strings.Repeat("-", widths[0])+" | "+strings.Repeat("-", widths[1])+" |")
		}
	}
	return strings.Join(lines, "\n")
}

func pad(cell string, width int) string {
	return cell + strings.Repeat(" ", width-utf8.RuneCountInString(cell))
}

// value: the catalog station ships, which every projection reads; a catalog
// that does not load stops the compiler.
func builtin() catalog.Catalog {
	c, err := catalog.Builtin()
	if err != nil {
		panic(err)
	}
	return c
}
