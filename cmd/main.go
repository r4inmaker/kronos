package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
	"github.com/r4inmaker/kronos/internal/agent"
	"github.com/r4inmaker/kronos/internal/browser"
	"github.com/r4inmaker/kronos/internal/logger"
)

func main() {
	godotenv.Load()
	agentLogger := logger.NewLogger("AGENT")

	ctx, cancel := browser.NewStealthBrowserContext(context.Background())
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	promptFile, err := os.Open("sysprompt.txt")
	if err != nil {
		log.Fatal(err)
	}
	promptBytes, err := io.ReadAll(promptFile)
	if err != nil {
		log.Fatal(err)
	}
	sysPrompt := fmt.Sprintf("%s", promptBytes)

	task := `
		Go ro Ryanair and find me the cheapest flight from Italy to Barcelona.
		Explore for the most optimal airport from Italy before clicking submit,
		make sure it exists in the dropdown.
		Book the flight. 2 cards, 5 day trip.
	`

	a := agent.NewAgent(ctx, cancel, "", task, sysPrompt, agentLogger)
	a.Run()
}
