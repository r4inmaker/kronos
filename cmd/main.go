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
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-infobars", true),
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
		Go to LinkedIn and log in using username: lizmanlizmanson@gmail.com and password: lizmajajca.
		Click the first post and then exit.
	`

	a := agent.NewAgent(ctx, cancel, "BORIS", task, sysPrompt, agentLogger)
	a.Run()
}
