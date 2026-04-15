package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/r4inmaker/kronos/internal/browser"
)

type CommandEngine struct {
	Browser       *browser.Browser
	Prompt        string
	ActionHistory []Action
	cancelFunc    context.CancelFunc
}

func NewCommandEngine(browser *browser.Browser, sysPrompt, task string, cancelFunc context.CancelFunc) *CommandEngine {
	initPrompt := fmt.Sprintf("[Objective]\n%s\n\n[Task]\n%s\n", sysPrompt, task)

	return &CommandEngine{
		Browser:       browser,
		Prompt:        initPrompt,
		ActionHistory: make([]Action, 0),
		cancelFunc:    cancelFunc,
	}
}

// Dispatch commands
func (ce *CommandEngine) DispatchCommands(resp *AgentResponse) (bool, error) {
	if len(resp.Actions) == 0 {
		return false, fmt.Errorf("agent responded with empty action slice")
	}

	exit := false
	for _, action := range resp.Actions {
		status, err := ce.DispatchCommand(action)
		if status {
			exit = true
		}
		if err != nil {
			return exit, err
		}
	}

	return exit, nil
}

// Browser command dispatch
func (ce *CommandEngine) DispatchCommand(action Action) (bool, error) {
	exit := false

	// Dispatch based on action name
	switch action.Name {

	// Navigate
	case "navigate":
		var params NavigateParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			return exit, err
		}
		if err := ce.Browser.Execute(ce.Browser.Navigate(params.URL)); err != nil {
			return exit, err
		}

		// Append and log
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Name:        "navigate",
			Description: action.Description,
			Reasoning:   action.Reasoning,
		})
		ce.Browser.Logger.Info(fmt.Sprintf("Action: [%s]\n\tReasoning: %q\n", action.Name, action.Reasoning))

	// Click
	case "click":
		var params ClickParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			return exit, err
		}
		description := fmt.Sprintf("clicked node [%d]", params.NodeID)
		if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
			description = fmt.Sprintf("clicked node %s", browser.FormatNode(node))
		}
		if err := ce.Browser.Execute(ce.Browser.ClickNode(params.NodeID)); err != nil {
			return exit, err
		}

		// Append and log
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Name:        "click",
			Description: description,
			Reasoning:   action.Reasoning,
		})
		ce.Browser.Logger.Info(fmt.Sprintf("Action: [%s]\n\tReasoning: %q\n", action.Name, action.Reasoning))

	// Send keys
	case "send_keys":
		var params SendKeysParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			return exit, err
		}
		description := fmt.Sprintf("sent keys %q to node [%d]", params.Keys, params.NodeID)
		if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
			description = fmt.Sprintf("sent keys %q to node %s", params.Keys, browser.FormatNode(node))
		}
		if err := ce.Browser.Execute(ce.Browser.SendKeysNode(params.NodeID, params.Keys)); err != nil {
			return exit, err
		}

		// Append and log
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Name:        "send_keys",
			Description: description,
			Reasoning:   action.Reasoning,
		})
		ce.Browser.Logger.Info(fmt.Sprintf("Action: [%s]\n\tReasoning: %q\n", action.Name, action.Reasoning))

	// Done
	case "done":
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Name:      "done",
			Reasoning: action.Reasoning,
		})

		ce.Browser.Logger.Info(fmt.Sprintf("Action: [%s]\n\tReasoning: %q\n", action.Name, action.Reasoning))
		ce.cancelFunc()
		return true, nil

	default:
		ce.cancelFunc()
		return true, fmt.Errorf("invalid browser command: %s", action.Name)
	}

	return exit, nil
}

// Utility functions
func (ce *CommandEngine) FormatActionHistory(max int) string {
	if len(ce.ActionHistory) == 0 {
		return "none"
	}
	history := ce.ActionHistory
	if len(history) > max {
		history = history[len(history)-max:]
	}

	var sb strings.Builder
	for _, a := range history {
		sb.WriteString(fmt.Sprintf("[%s] (%s | Reasoning: %q)\n",
			a.Name, a.Description, a.Reasoning))
	}
	return sb.String()
}

func nodeID(id int64) accessibility.NodeID {
	return accessibility.NodeID(strconv.Itoa(int(id)))
}
