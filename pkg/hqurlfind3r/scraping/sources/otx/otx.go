package otx

import (
	"encoding/json"
	"fmt"

	"github.com/hueristiq/hqurlfind3r/pkg/hqurlfind3r/scraping"
	"github.com/hueristiq/hqurlfind3r/pkg/hqurlfind3r/session"
)

type Source struct{}

type response struct {
	HasNext    bool `json:"has_next"`
	ActualSize int  `json:"actual_size"`
	URLList    []struct {
		Domain   string `json:"domain"`
		URL      string `json:"url"`
		Hostname string `json:"hostname"`
		HTTPCode int    `json:"httpcode"`
		PageNum  int    `json:"page_num"`
		FullSize int    `json:"full_size"`
		Paged    bool   `json:"paged"`
	} `json:"url_list"`
}

func (source *Source) Run(domain string, ses *session.Session, includeSubs bool) (URLs chan scraping.URL) {
	URLs = make(chan scraping.URL)

	go func() {
		defer close(URLs)

		for page := 0; ; page++ {
			res, err := ses.SimpleGet(fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=%d&page=%d", domain, 200, page))
			if err != nil {
				return
			}

			var results response

			if err := json.Unmarshal(res.Body(), &results); err != nil {
				return
			}

			for _, i := range results.URLList {
				if URL, ok := scraping.NormalizeURL(i.URL, ses.Scope); ok {
					URLs <- scraping.URL{Source: source.Name(), Value: URL}
				}
			}

			if !results.HasNext {
				break
			}
		}
	}()

	return
}

func (source *Source) Name() string {
	return "otx"
}
