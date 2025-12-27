package main

import (
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/joho/godotenv"

	"github.com/Nehilsa2/linkedin_automation/auth"
	"github.com/Nehilsa2/linkedin_automation/search"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Unable to load .env file; falling back to existing environment variables")
	}

	u := launcher.New().
		Leakless(false).
		Headless(false).
		MustLaunch()

	browser := rod.New().
		ControlURL(u).
		MustConnect()

	defer browser.MustClose()

	err := auth.EnsureAuthenticated(browser)
	if err != nil {
		log.Fatal("❌ Authentication failed:", err)
	}
	people, _ := search.FindPeople(browser, "software engineer")
	fmt.Println(people)

	companies, _ := search.FindCompanies(browser, "fintech")
	fmt.Println(companies)

}
