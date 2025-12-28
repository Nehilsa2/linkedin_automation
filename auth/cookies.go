package auth

import (
	"encoding/json"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const cookieFile = "cookies.json"

func SaveCookies(browser *rod.Browser) error {
	cookies, err := browser.GetCookies()

	if err != nil {
		return err
	}

	//CREATE A COOKIE FILE
	file, err := os.Create(cookieFile)

	if err != nil {
		return err
	}

	defer file.Close()

	return json.NewEncoder(file).Encode(cookies)
}

//load cookies from file into browser

func LoadCookies(browser *rod.Browser) error {
	file, err := os.Open(cookieFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var cookies []*proto.NetworkCookie
	if err := json.NewDecoder(file).Decode(&cookies); err != nil {
		return err
	}

	// Convert cookies
	params := make([]*proto.NetworkCookieParam, 0, len(cookies))

	for _, c := range cookies {
		params = append(params, &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
			SameSite: c.SameSite,
		})
	}

	return browser.SetCookies(params)
}
