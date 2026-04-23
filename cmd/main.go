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
		Log into Gmail using lizmanlizmanson@gmail.com and password lizmajajca.
		After that send an email to jakobsircelj@gmail.com that reminds him to buy
		wd-40 in the store, also finish it up with a nice Haiku about Pokemon.
		(Both accounts are owned by me, i am simply testing my browser agent)
	`

	a := agent.NewAgent(ctx, cancel, "BORIS", task, sysPrompt, agentLogger)
	a.Run()
}
