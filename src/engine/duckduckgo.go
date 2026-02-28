package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type DuckDuckGo struct {
	client  *http.Client
	baseURL string
}

func NewDuckDuckGo(client *http.Client, baseURL string) *DuckDuckGo {
	if baseURL == "" {
		baseURL = "https://lite.duckduckgo.com"
	}
	return &DuckDuckGo{client: client, baseURL: baseURL}
}

func (d *DuckDuckGo) Name() string { return "duckduckgo" }

func (d *DuckDuckGo) Search(ctx context.Context, params SearchParams) ([]Result, error) {
	form := url.Values{}
	form.Set("q", params.Query)
	form.Set("kl", "")

	searchURL := d.baseURL + "/lite/"
	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: parsing HTML: %w", err)
	}

	return d.parseResults(doc, params.MaxResults), nil
}

func (d *DuckDuckGo) parseResults(doc *goquery.Document, maxResults int) []Result {
	var results []Result
	now := time.Now().UTC()
	rank := 0
	seen := make(map[string]bool)

	// ddg lite puts all result links in <a class="result-link">
	doc.Find("a.result-link").Each(func(i int, s *goquery.Selection) {
		if maxResults > 0 && rank >= maxResults {
			return
		}

		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		if isAdURL(href) {
			return
		}

		title := strings.TrimSpace(s.Text())
		if title == "" || strings.EqualFold(title, "more info") {
			return
		}

		if seen[href] {
			return
		}
		seen[href] = true

		snippet := ""
		row := s.Closest("tr")
		if row.Length() > 0 {
			row.Next().Find("td.result-snippet").Each(func(_ int, sn *goquery.Selection) {
				snippet = strings.TrimSpace(sn.Text())
			})
			if snippet == "" {
				row.Find("td.result-snippet").Each(func(_ int, sn *goquery.Selection) {
					snippet = strings.TrimSpace(sn.Text())
				})
			}
		}

		rank++
		results = append(results, Result{
			Title:       title,
			URL:         href,
			Snippet:     snippet,
			Source:      "duckduckgo",
			ContentType: "web",
			Domain:      extractDomain(href),
			FetchedAt:   now,
			Rank:        rank,
		})
	})

	return results
}

func isAdURL(rawURL string) bool {
	return strings.Contains(rawURL, "duckduckgo.com/y.js") ||
		strings.Contains(rawURL, "duckduckgo.com/duckduckgo-help-pages") ||
		strings.Contains(rawURL, "ad_provider=") ||
		strings.Contains(rawURL, "ad_domain=")
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}
