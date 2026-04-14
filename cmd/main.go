package main

import (
	"context"

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

	sysPrompt := `You are a browser automation agent. You control a real web browser.

	On each cycle you receive:
	- [Browser State]: accessibility tree of the current page (role, name, node_id)
	- [Action History]: what you have done so far and the results

	You must respond with JSON matching this exact schema:
	{
		"reasoning": "your step by step thinking for provided action(s)",
		"actions": [
			{ "type": "browser"|"agent", "name": "...", "params": { ... }, "reasoning" }
		]
	}

	Each action must include its own short reasoning, so that command history is consistent.

	BROWSER ACTIONS (type: "browser"):
		navigate    { "url": string }
		click       { "node_id": number }
		send_keys   { "node_id": number, "keys": string, "simulate": bool }
								simulate=true for OTP/character-by-character inputs

	AGENT ACTIONS (type: "agent"):
		done            {}

	RULES:
	- Only use node_ids visible in the current browser state, never guess
	- Actions execute in order, stop on first failure
	- Use update_history to summarize completed sub-goals or correct stale entries
	- Always emit done when the task is complete, or if you find yourself in an unrecoverable cycle
	- You are responsible for providing reasoning to actions, which will be autoregressively fed into
	  you on the next cycle, which means you need to provide reasoning for a action(s) as well as for
	  each individual action. Be verobose so you dont get stuck in a cycle but concise enough to be
	  mindful of your token limit.
	`

	task := `
		Go to LinkedIn and log in using username: lizmanlizmanson@gmail.com and password: lizmajajca.
		Give me the summary of the first post that you find (what the text says).
	`

	a := agent.NewAgent(ctx, "BORIS", task, sysPrompt, agentLogger)
	a.Run()

	// b := browser.NewBrowser(ctx, agentLogger)
	// b.Execute(
	// 	b.Navigate("https://linkedin.com"),
	// 	b.WaitReady("body"),
	// )
	// b.BuildTree()
	// agentLogger.Info(b.SprintTree(
	// 	browser.FilterNodeDefault(),
	// ))
}
