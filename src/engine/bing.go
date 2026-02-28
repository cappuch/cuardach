package engine

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

type Bing struct {
	baseURL string
}

func NewBing(baseURL string) *Bing {
	if baseURL == "" {
		baseURL = "https://www.bing.com"
	}
	return &Bing{baseURL: baseURL}
}

func (b *Bing) Name() string { return "bing" }

func (b *Bing) Search(ctx context.Context, params SearchParams) ([]Result, error) {
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}

	browser, err := getBrowser()
	if err != nil {
		return nil, fmt.Errorf("bing: %w", err)
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("bing: creating page: %w", err)
	}
	defer page.Close()

	searchURL := fmt.Sprintf("%s/search?q=%s&count=%d&setlang=en&cc=US&mkt=en-US",
		b.baseURL, url.QueryEscape(params.Query), maxResults)
	if params.Page > 1 {
		searchURL += fmt.Sprintf("&first=%d", (params.Page-1)*10+1)
	}

	if err := page.Navigate(searchURL); err != nil {
		return nil, fmt.Errorf("bing: navigating: %w", err)
	}
	page.Timeout(15 * time.Second)
	if err := page.WaitDOMStable(time.Second, 0.1); err != nil {
		// Not fatal — try extracting anyway
	}

	jsResults, err := page.Eval(`() => {
		function decodeBingURL(href) {
			// Bing wraps URLs as bing.com/ck/a?...&u=a1<base64>&...
			try {
				const url = new URL(href);
				const u = url.searchParams.get('u');
				if (u && u.startsWith('a1')) {
					return atob(u.substring(2));
				}
			} catch {}
			return href;
		}
		const results = [];
		const seen = new Set();
		const blocks = document.querySelectorAll('li.b_algo');
		for (const block of blocks) {
			const h2 = block.querySelector('h2');
			if (!h2) continue;
			const link = h2.querySelector('a[href]');
			if (!link) continue;
			let href = link.getAttribute('href') || '';
			if (!href) continue;
			// Bing ad clicks use /aclick
			if (href.includes('/aclick')) continue;
			// Decode the bing.com/ck/a redirect to get actual URL
			if (href.includes('bing.com/ck/')) {
				href = decodeBingURL(href);
			}
			if (!href || href.includes('bing.com') || href.includes('microsoft.com')) continue;
			if (seen.has(href)) continue;
			seen.add(href);
			const title = link.textContent.trim();
			if (!title) continue;
			let snippet = '';
			const cap = block.querySelector('.b_caption p, p.b_lineclamp2, p.b_lineclamp3, p.b_lineclamp4, .b_algoSlug');
			if (cap) snippet = cap.textContent.trim();
			if (!snippet) {
				const p = block.querySelector('p');
				if (p) snippet = p.textContent.trim();
			}
			results.push({url: href, title: title, snippet: snippet});
		}
		return results;
	}`)
	if err != nil {
		return nil, fmt.Errorf("bing: extracting results: %w", err)
	}

	now := time.Now().UTC()
	var results []Result
	for i, item := range jsResults.Value.Arr() {
		if i >= maxResults {
			break
		}
		obj := item.Map()
		rawURL := obj["url"].Str()
		title := obj["title"].Str()
		snippet := ""
		if s, ok := obj["snippet"]; ok {
			snippet = s.Str()
		}

		if isBingInternalURL(rawURL) {
			continue
		}

		results = append(results, Result{
			Title:       title,
			URL:         rawURL,
			Snippet:     snippet,
			Source:      "bing",
			ContentType: "web",
			Domain:      extractDomain(rawURL),
			FetchedAt:   now,
			Rank:        len(results) + 1,
		})
	}

	return results, nil
}

func isBingInternalURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return strings.HasSuffix(host, "bing.com") ||
		strings.HasSuffix(host, "microsoft.com") ||
		strings.HasSuffix(host, "msn.com")
}
