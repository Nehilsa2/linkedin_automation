// package search

// import (
// 	"fmt"
// 	"net/url"
// 	"strings"
// 	"time"

// 	"github.com/go-rod/rod"
// )

// // FindPeople searches LinkedIn people results and paginates safely
// func FindPeople(browser *rod.Browser, keyword string, maxPages int) ([]string, error) {

// 	searchURL := "https://www.linkedin.com/search/results/people/?keywords=" +
// 		url.QueryEscape(keyword)

// 	page := browser.MustPage(searchURL)
// 	time.Sleep(4 * time.Second) // allow initial render

// 	var allLinks []string
// 	seen := make(map[string]bool)

// 	for pageNum := 1; pageNum <= maxPages; pageNum++ {

// 		// 🔍 Extract visible profile links (NO blocking waits)
// 		anchors, _ := page.Elements(`a[href^="https://www.linkedin.com/in/"]`)
// 		pageLinks := 0

// 		for _, a := range anchors {
// 			href, _ := a.Attribute("href")
// 			if href == nil {
// 				continue
// 			}

// 			link := strings.Split(*href, "?")[0]
// 			if !seen[link] {
// 				seen[link] = true
// 				allLinks = append(allLinks, link)
// 				pageLinks++
// 			}
// 		}

// 		fmt.Printf("👤 Page %d → %d profiles\n", pageNum, pageLinks)

// 		// 🔚 Try to paginate
// 		nextBtn, err := page.Element(
// 			`button[data-testid="pagination-controls-next-button"]`,
// 		)
// 		if err != nil {
// 			fmt.Println("ℹ️ No Next button found, stopping")
// 			break
// 		}

// 		if d, _ := nextBtn.Attribute("disabled"); d != nil {
// 			fmt.Println("ℹ️ Next button disabled, stopping")
// 			break
// 		}

// 		// Scroll + click (React-safe)
// 		nextBtn.MustScrollIntoView()
// 		time.Sleep(800 * time.Millisecond)

// 		fmt.Println("➡️ Clicking Next page")
// 		nextBtn.MustClick()

// 		// React re-render (not navigation)
// 		time.Sleep(5 * time.Second)
// 	}

//		return allLinks, nil
//	}
package search

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

func FindPeople(browser *rod.Browser, keyword string, maxPages int) ([]string, error) {

	searchURL := "https://www.linkedin.com/search/results/people/?keywords=" +
		url.QueryEscape(keyword)

	page := browser.MustPage(searchURL)
	time.Sleep(4 * time.Second)

	var allLinks []string
	seen := make(map[string]bool)

	for pageNum := 1; pageNum <= maxPages; pageNum++ {

		// Extract profiles (even if zero)
		pageLinks := 0
		anchors, _ := page.Elements(`a[href^="https://www.linkedin.com/in/"]`)

		for _, a := range anchors {
			href, _ := a.Attribute("href")
			if href == nil {
				continue
			}

			link := strings.Split(*href, "?")[0]
			if !seen[link] {
				seen[link] = true
				allLinks = append(allLinks, link)
				pageLinks++
			}
		}

		fmt.Printf("👤 Page %d → %d profiles\n", pageNum, pageLinks)

		// Use shared pagination function
		hasNext, _ := ClickNextPage(page)
		if !hasNext {
			break
		}
	}

	return allLinks, nil
}
