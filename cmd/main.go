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
	//ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	//defer cancel()

	sysPrompt := `You are a browser automation agent. You control a real web browser.

	Each cycle you receive:
	- [Browser State]: accessibility tree of the current page (role, name, node_id)
	- [Action History]: what you have done so far and the results

	You must respond with JSON matching this exact schema:
	{
		"reasoning": "your step by step thinking",
		"actions": [
			{ "type": "browser"|"agent", "name": "...", "params": { ... } }
		]
	}

	BROWSER ACTIONS (type: "browser"):
		navigate    { "url": string }
		click       { "node_id": number }
		send_keys   { "node_id": number, "keys": string, "simulate": bool }
								simulate=true for OTP/character-by-character inputs

	AGENT ACTIONS (type: "agent"):
		update_history  { "index": -1|number, "action": { "action": string, "result": string } }
										index=-1 to append, index=N to overwrite entry N
		done            {}

	RULES:
	- Only use node_ids visible in the current browser state, never guess
	- Actions execute in order, stop on first failure
	- Use update_history to summarize completed sub-goals or correct stale entries
	- Always emit done when the task is complete`

	task := `
		Go to LinkedIn and log in using username: lizmanlizmanson@gmail.com and password: lizmajajca.
		Give me the summary of the first post that you find (what the text says).
	`

	a := agent.NewAgent(ctx, "BORIS", task, sysPrompt, agentLogger)
	a.Run()
}
