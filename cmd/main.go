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
	"github.com/r4inmaker/kronos/internal/logger"
)

func main() {
	godotenv.Load()
	agentLogger := logger.NewLogger("AGENT")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		// Use a string instead of a slice to avoid the "invalid flag" error
		chromedp.Flag("excludeSwitches", "enable-automation"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
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
		lotion in the store, also finish it up with a nice Haiku about Boris the Gator.
		(Both accounts are owned by me, i am simply testing my browser agent)
	`

	a := agent.NewAgent(ctx, cancel, "BORIS", task, sysPrompt, agentLogger)
	a.Run()
}
