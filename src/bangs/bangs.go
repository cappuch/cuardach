package bangs

import (
	"net/url"
	"strings"
)

type Bang struct {
	Name    string // short trigger, e.g. "g"
	Aliases []string
	Title   string // human-readable name
	URL     string // search URL with %s placeholder for query
}

type Result struct {
	Bang     *Bang  // matched bang, nil if none
	Query    string // remaining query with bang stripped
	Redirect string // full redirect URL if bang matched
}

var registry = map[string]*Bang{}

func init() {
	for i := range defaults {
		b := &defaults[i]
		registry[b.Name] = b
		for _, a := range b.Aliases {
			registry[a] = b
		}
	}
}

func Parse(query string) Result {
	words := strings.Fields(query)
	if len(words) == 0 {
		return Result{Query: query}
	}

	if strings.HasPrefix(words[0], "!") {
		key := strings.ToLower(strings.TrimPrefix(words[0], "!"))
		if b, ok := registry[key]; ok {
			q := strings.Join(words[1:], " ")
			return Result{
				Bang:     b,
				Query:    q,
				Redirect: buildURL(b.URL, q),
			}
		}
	}

	last := words[len(words)-1]
	if strings.HasPrefix(last, "!") {
		key := strings.ToLower(strings.TrimPrefix(last, "!"))
		if b, ok := registry[key]; ok {
			q := strings.Join(words[:len(words)-1], " ")
			return Result{
				Bang:     b,
				Query:    q,
				Redirect: buildURL(b.URL, q),
			}
		}
	}

	return Result{Query: query}
}

func List() []Bang {
	seen := map[string]bool{}
	var out []Bang
	for _, b := range defaults {
		if !seen[b.Name] {
			seen[b.Name] = true
			out = append(out, b)
		}
	}
	return out
}

func buildURL(tmpl, query string) string {
	return strings.ReplaceAll(tmpl, "%s", url.QueryEscape(query))
}

func IsEngineBang(r Result) (string, bool) {
	if r.Bang == nil {
		return "", false
	}
	if strings.HasPrefix(r.Redirect, "engine://") {
		return strings.TrimPrefix(r.Redirect, "engine://"), true
	}
	return "", false
}

var defaults = []Bang{
	// engine selection
	{Name: "d", Aliases: []string{"ddg", "duckduckgo"}, Title: "DuckDuckGo", URL: "engine://duckduckgo"},
	{Name: "b", Aliases: []string{"bing"}, Title: "Bing", URL: "engine://bing"},

	{Name: "g", Aliases: []string{"google"}, Title: "Google", URL: "https://www.google.com/search?q=%s"},
	{Name: "sp", Aliases: []string{"startpage"}, Title: "Startpage", URL: "https://www.startpage.com/do/dsearch?query=%s"},
	{Name: "sx", Aliases: []string{"searx"}, Title: "SearXNG", URL: "https://searx.be/search?q=%s"},
	{Name: "w", Aliases: []string{"wiki", "wikipedia"}, Title: "Wikipedia", URL: "https://en.wikipedia.org/w/index.php?search=%s"},
	{Name: "wt", Aliases: []string{"wiktionary"}, Title: "Wiktionary", URL: "https://en.wiktionary.org/w/index.php?search=%s"},
	{Name: "gh", Aliases: []string{"github"}, Title: "GitHub", URL: "https://github.com/search?q=%s"},
	{Name: "gl", Aliases: []string{"gitlab"}, Title: "GitLab", URL: "https://gitlab.com/search?search=%s"},
	{Name: "so", Aliases: []string{"stackoverflow"}, Title: "Stack Overflow", URL: "https://stackoverflow.com/search?q=%s"},
	{Name: "cr", Aliases: []string{"crates"}, Title: "crates.io", URL: "https://crates.io/search?q=%s"},
	{Name: "npm", Title: "npm", URL: "https://www.npmjs.com/search?q=%s"},
	{Name: "pypi", Aliases: []string{"pip"}, Title: "PyPI", URL: "https://pypi.org/search/?q=%s"},
	{Name: "pkg", Aliases: []string{"gopkg"}, Title: "Go Packages", URL: "https://pkg.go.dev/search?q=%s"},
	{Name: "doc", Aliases: []string{"godoc"}, Title: "Go Docs", URL: "https://pkg.go.dev/%s"},
	{Name: "yt", Aliases: []string{"youtube"}, Title: "YouTube", URL: "https://www.youtube.com/results?search_query=%s"},
	{Name: "r", Aliases: []string{"reddit"}, Title: "Reddit", URL: "https://www.reddit.com/search/?q=%s"},
	{Name: "hn", Aliases: []string{"hackernews"}, Title: "Hacker News", URL: "https://hn.algolia.com/?q=%s"},
	{Name: "tw", Aliases: []string{"twitter", "x"}, Title: "X/Twitter", URL: "https://x.com/search?q=%s"},
	{Name: "m", Aliases: []string{"maps", "osm"}, Title: "OpenStreetMap", URL: "https://www.openstreetmap.org/search?query=%s"},
	{Name: "tr", Aliases: []string{"translate"}, Title: "DeepL", URL: "https://www.deepl.com/translator#auto/en/%s"},
	{Name: "az", Aliases: []string{"amazon"}, Title: "Amazon", URL: "https://www.amazon.com/s?k=%s"},
	{Name: "imdb", Title: "IMDb", URL: "https://www.imdb.com/find?q=%s"},
	{Name: "arch", Aliases: []string{"archwiki"}, Title: "Arch Wiki", URL: "https://wiki.archlinux.org/index.php?search=%s"},
	{Name: "man", Title: "man pages", URL: "https://man.archlinux.org/search?q=%s"},
}
