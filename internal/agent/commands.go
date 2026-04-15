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
		cancelFunc: 	 cancelFunc,	
	}
}

// Dispatch commands
func (ce *CommandEngine) DispatchCommands(resp *AgentResponse) (bool, error) {
	if len(resp.Actions) == 0 {
		return false, fmt.Errorf("agent responded with empty action slice")
	}

	for _, action := range resp.Actions {
		switch action.Type {
		case "browser":
			err := ce.DispatchBrowserCommand(action)
			if err != nil {
				return false, err
			}
			ce.Browser.Logger.Info(fmt.Sprintf("%s-action: [%s]\n\tReasoning: %q\n", action.Type, action.Name, action.Reasoning))

		case "agent":
			exit, err := ce.DispatchAgentCommand(action)
			if err != nil {
				return false, err
			}
			if exit {
				return true, err
			}
			ce.Browser.Logger.Info(fmt.Sprintf("%s-action: [%s]\n\tReasoning: %q\n", action.Type, action.Name, action.Reasoning))

		default:
			return false, fmt.Errorf("unknown action type: %s", action.Type)
		}
	}

	return false, nil
}

// Browser command dispatch
func (ce *CommandEngine) DispatchBrowserCommand(action Action) error {
	switch action.Name {

	case "navigate":
		var params NavigateParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			return err
		}

		// Execute action
		var result string
		if err := ce.Browser.Execute(ce.Browser.Navigate(params.URL)); err != nil {
			result = fmt.Sprintf("failure, error: %v", err)
		} else {
			result = "success"
		}

		// Append to history
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Type:        "browser",
			Name:        "navigate",
			Description: action.Description,
			Reasoning:   action.Reasoning,
			Result:      result,
		})

	case "click":
		var params ClickParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			return err
		}

		// Description
		description := fmt.Sprintf("clicked node [%s]", params.NodeID)
		if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
			description = fmt.Sprintf("clicked node %s",
				browser.FormatNode(node),
			)
		}

		// Execute action
		var result string
		if err := ce.Browser.Execute(ce.Browser.ClickNode(params.NodeID)); err != nil {
			result = fmt.Sprintf("failure, error: %v", err)
		} else {
			result = "success"
		}

		// Append to history
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Type:        "browser",
			Name:        "click",
			Description: description,
			Reasoning:   action.Reasoning,
			Result:      result,
		})

	case "send_keys":
		var params SendKeysParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			return err
		}

		// Description
		description := fmt.Sprintf("send keys %q to node [%s]", params.Keys, params.NodeID)
		if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
			description = fmt.Sprintf("sent keys %q to node %s",
				params.Keys,
				browser.FormatNode(node),
			)
		}

		// Execute action
		var result string
		if err := ce.Browser.Execute(ce.Browser.SendKeysNode(params.NodeID, params.Keys, params.Simulate)); err != nil {
			result = fmt.Sprintf("failed sending keys %q to node, error: %v", params.Keys, err)
		} else {
			result = "success"
		}

		// Append to history
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Type:        "browser",
			Name:        "send_keys",
			Description: description,
			Reasoning:   action.Reasoning,
			Result:      result,
		})

	default:
		return fmt.Errorf("invalid browser command: %s", action.Name)
	}

	return nil
}

// Agent command dispatch
func (ce *CommandEngine) DispatchAgentCommand(action Action) (bool, error) {
	switch action.Name {
	case "done":
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Type: "agent",
			Name: "done",
			Reasoning: action.Reasoning,
		})
		ce.cancelFunc()
		return true, nil

	default:
		return false, fmt.Errorf("invalid agent command : %s", action.Name)
	}
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
		sb.WriteString(fmt.Sprintf("[%s] (%s Reasoning: %q) → %s\n",
			a.Name, a.Description, a.Reasoning, a.Result))
	}
	return sb.String()
}

func nodeID(id int64) accessibility.NodeID {
	return accessibility.NodeID(strconv.Itoa(int(id)))
}
