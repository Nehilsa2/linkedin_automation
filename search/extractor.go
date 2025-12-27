// extract target url from the result card
package search

import (
	"strings"

	"github.com/go-rod/rod"
)

func ExtractProfileLinks(page *rod.Page) ([]string, error) {
	anchors := page.MustElements(`a.app-aware-link`)

	unique := make(map[string]bool)
	var results []string

	for _, a := range anchors {
		href, err := a.Attribute("href")
		if err != nil || href == nil {
			continue
		}

		link := *href

		// People or Company URLs only
		if strings.Contains(link, "/in/") || strings.Contains(link, "/company/") {
			// Clean tracking params
			link = strings.Split(link, "?")[0]

			if !unique[link] {
				unique[link] = true
				results = append(results, link)
			}
		}
	}
	return results, nil

}
