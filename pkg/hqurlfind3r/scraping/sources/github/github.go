package github

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hueristiq/hqurlfind3r/pkg/hqurlfind3r/scraping"
	"github.com/hueristiq/hqurlfind3r/pkg/hqurlfind3r/session"
	"github.com/tomnomnom/linkheader"
	"github.com/valyala/fasthttp"
)

type Source struct{}

type textMatch struct {
	Fragment string `json:"fragment"`
}

type item struct {
	Name        string      `json:"name"`
	HTMLURL     string      `json:"html_url"`
	TextMatches []textMatch `json:"text_matches"`
}

type response struct {
	TotalCount int    `json:"total_count"`
	Items      []item `json:"items"`
}

func (source *Source) Run(domain string, ses *session.Session, includeSubs bool) chan scraping.URL {
	URLs := make(chan scraping.URL)

	go func() {
		defer close(URLs)

		if len(ses.Keys.GitHub) == 0 {
			return
		}

		tokens := NewTokenManager(ses.Keys.GitHub)

		searchURL := fmt.Sprintf("https://api.github.com/search/code?per_page=100&q=%s&sort=created&order=asc", domain)
		source.Enumerate(searchURL, domainRegexp(domain, includeSubs), tokens, ses, URLs)
	}()

	return URLs
}

func (source *Source) Enumerate(searchURL string, domainRegexp *regexp.Regexp, tokens *Tokens, ses *session.Session, URLs chan scraping.URL) {
	token := tokens.Get()

	if token.RetryAfter > 0 {
		if len(tokens.pool) == 1 {
			time.Sleep(time.Duration(token.RetryAfter) * time.Second)
		} else {
			token = tokens.Get()
		}
	}

	res, err := ses.Request(
		fasthttp.MethodGet,
		searchURL,
		"",
		map[string]string{
			"Accept":        "application/vnd.github.v3.text-match+json",
			"Authorization": "token " + token.Hash,
		},
		nil,
	)
	isForbidden := res != nil && res.StatusCode() == fasthttp.StatusForbidden
	if err != nil && !isForbidden {
		return
	}

	ratelimitRemaining, _ := strconv.ParseInt(string(res.Header.Peek("X-Ratelimit-Remaining")), 10, 64)
	if isForbidden && ratelimitRemaining == 0 {
		retryAfterSeconds, _ := strconv.ParseInt(string(res.Header.Peek("Retry-After")), 10, 64)
		tokens.setCurrentTokenExceeded(retryAfterSeconds)

		source.Enumerate(searchURL, domainRegexp, tokens, ses, URLs)
	}

	var results response

	if err := json.Unmarshal(res.Body(), &results); err != nil {
		return
	}

	err = proccesItems(results.Items, domainRegexp, source.Name(), ses, URLs)
	if err != nil {
		return
	}

	linksHeader := linkheader.Parse(string(res.Header.Peek("Link")))

	for _, link := range linksHeader {
		if link.Rel == "next" {
			nextURL, err := url.QueryUnescape(link.URL)
			if err != nil {
				return
			}
			source.Enumerate(nextURL, domainRegexp, tokens, ses, URLs)
		}
	}
}

func proccesItems(items []item, domainRegexp *regexp.Regexp, name string, ses *session.Session, URLs chan scraping.URL) error {
	for _, item := range items {
		res, _ := ses.SimpleGet(rawContentURL(item.HTMLURL))

		if res.StatusCode() == fasthttp.StatusOK {
			scanner := bufio.NewScanner(bytes.NewReader(res.Body()))
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				for _, URL := range domainRegexp.FindAllString(normalizeContent(line), -1) {
					if URL, ok := scraping.NormalizeURL(URL, ses.Scope); ok {
						URLs <- scraping.URL{Source: name, Value: URL}
					}
				}
			}
		}

		for _, textMatch := range item.TextMatches {
			for _, URL := range domainRegexp.FindAllString(normalizeContent(textMatch.Fragment), -1) {
				if URL, ok := scraping.NormalizeURL(URL, ses.Scope); ok {
					URLs <- scraping.URL{Source: name, Value: URL}
				}
			}
		}
	}
	return nil
}

func normalizeContent(content string) string {
	content, _ = url.QueryUnescape(content)
	content = strings.ReplaceAll(content, "\\t", "")
	content = strings.ReplaceAll(content, "\\n", "")
	return content
}

func rawContentURL(URL string) string {
	URL = strings.ReplaceAll(URL, "https://github.com/", "https://raw.githubusercontent.com/")
	URL = strings.ReplaceAll(URL, "/blob/", "/")
	return URL
}

func domainRegexp(host string, includeSubs bool) (URLRegex *regexp.Regexp) {
	URLRegex = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`)
	return URLRegex
}

func (source *Source) Name() string {
	return "github"
}
