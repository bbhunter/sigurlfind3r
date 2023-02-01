package intelx

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hueristiq/hqurlfind3r/pkg/hqurlfind3r/scraping"
	"github.com/hueristiq/hqurlfind3r/pkg/hqurlfind3r/session"
)

type searchResponseType struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type selectorType struct {
	Selectvalue string `json:"selectorvalue"`
}

type searchResultType struct {
	Selectors []selectorType `json:"selectors"`
	Status    int            `json:"status"`
}

type requestBody struct {
	Term       string        `json:"term"`
	Timeout    time.Duration `json:"timeout"`
	MaxResults int           `json:"maxresults"`
	Media      int           `json:"media"`
}

type Source struct{}

func (source *Source) Run(domain string, ses *session.Session, includeSubs bool) (URLs chan scraping.URL) {
	URLs = make(chan scraping.URL)

	go func() {
		defer close(URLs)

		if ses.Keys.IntelXKey == "" || ses.Keys.IntelXHost == "" {
			return
		}

		searchURL := fmt.Sprintf("https://%s/phonebook/search?k=%s", ses.Keys.IntelXHost, ses.Keys.IntelXKey)
		reqBody := requestBody{
			Term:       domain,
			MaxResults: 100000,
			Media:      0,
			Timeout:    20,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return
		}

		res, err := ses.SimplePost(searchURL, "application/json", body)
		if err != nil {
			return
		}

		var response searchResponseType

		if err := json.Unmarshal(res.Body(), &response); err != nil {
			return
		}

		resultsURL := fmt.Sprintf("https://%s/phonebook/search/result?k=%s&id=%s&limit=10000", ses.Keys.IntelXHost, ses.Keys.IntelXKey, response.ID)
		status := 0

		for status == 0 || status == 3 {
			res, err = ses.Get(resultsURL, "", nil)
			if err != nil {
				return
			}

			var response searchResultType

			if err := json.Unmarshal(res.Body(), &response); err != nil {
				return
			}

			status = response.Status
			for _, hostname := range response.Selectors {
				if URL, ok := scraping.NormalizeURL(hostname.Selectvalue, ses.Scope); ok {
					URLs <- scraping.URL{Source: source.Name(), Value: URL}
				}
			}
		}
	}()

	return URLs
}

func (source *Source) Name() string {
	return "intelx"
}
