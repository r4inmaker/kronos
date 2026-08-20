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
		Go to mimovrste.com and find the cheapest variant
		of Beko BDIN38561P washing machine. 
		Add it to cart, proceed to checkout 
		and then quit before you purchase it.
		I just want to see if you can do it.
	`

	a := agent.NewAgent(ctx, cancel, "", task, sysPrompt, agentLogger)
	a.Run()
}
