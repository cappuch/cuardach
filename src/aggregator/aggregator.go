package aggregator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/cappuch/cuardach/src/engine"
)

type AggregatedResult struct {
	engine.Result
	Sources    []string `json:"sources"`
	Score      float64  `json:"score"`
}

type Aggregator struct {
	engines []engine.Engine
}

func New(engines []engine.Engine) *Aggregator {
	return &Aggregator{engines: engines}
}

func (a *Aggregator) Search(ctx context.Context, params engine.SearchParams) ([]AggregatedResult, []error) {
	type engineResult struct {
		results []engine.Result
		err     error
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	allResults := make([]engine.Result, 0)
	var errs []error

	for _, eng := range a.engines {
		wg.Add(1)
		go func(e engine.Engine) {
			defer wg.Done()
			results, err := e.Search(ctx, params)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
				return
			}
			allResults = append(allResults, results...)
		}(eng)
	}

	wg.Wait()

	merged := deduplicate(allResults)
	rank(merged)

	if params.MaxResults > 0 && len(merged) > params.MaxResults {
		merged = merged[:params.MaxResults]
	}

	return merged, errs
}

// deduplicate merges results with the same normalized URL.
func deduplicate(results []engine.Result) []AggregatedResult {
	seen := make(map[string]*AggregatedResult)
	var order []string

	for _, r := range results {
		key := urlHash(r.URL)
		if existing, ok := seen[key]; ok {
			// Merge: add source, keep best snippet
			if !containsStr(existing.Sources, r.Source) {
				existing.Sources = append(existing.Sources, r.Source)
			}
			if len(r.Snippet) > len(existing.Snippet) {
				existing.Snippet = r.Snippet
			}
			// Accumulate RRF score: 1/(k+rank) with k=60
			existing.Score += 1.0 / float64(60+r.Rank)
		} else {
			agg := &AggregatedResult{
				Result:  r,
				Sources: []string{r.Source},
				Score:   1.0 / float64(60+r.Rank),
			}
			seen[key] = agg
			order = append(order, key)
		}
	}

	merged := make([]AggregatedResult, 0, len(order))
	for _, key := range order {
		merged = append(merged, *seen[key])
	}
	return merged
}

// rank sorts results by composite RRF score descending.
func rank(results []AggregatedResult) {
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// normalizeURL normalizes a URL for deduplication.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Host = strings.TrimPrefix(u.Host, "www.")
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")

	// Strip tracking parameters
	q := u.Query()
	for key := range q {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") ||
			lower == "fbclid" || lower == "gclid" ||
			lower == "ref" || lower == "source" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func urlHash(rawURL string) string {
	normalized := normalizeURL(rawURL)
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
