# Kronos

A browser agent that carries out a task you give it in plain English.

## Setup

1. Add your OpenAI API key to `.env`:
   ```
   OPENAI_API_KEY=your-key-here
   ```
2. Open `cmd/main.go` and change the `task` string to whatever you want the agent to do.
3. Adjust anything else in `cmd/main.go` (browser options, agent name, etc.) as needed.
4. Run it:
   ```
   go run cmd/main.go
   ```
