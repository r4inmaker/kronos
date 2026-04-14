package agent

import (
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
}

func NewCommandEngine(browser *browser.Browser, sysPrompt, task string) *CommandEngine {
	initPrompt := fmt.Sprintf("[Objective]\n%s\n\n[Task]\n%s\n", sysPrompt, task)

	return &CommandEngine{
		Browser:       browser,
		Prompt:        initPrompt,
		ActionHistory: make([]Action, 0),
	}
}

// Dispatch commands
func (ce *CommandEngine) DispatchCommands(resp *AgentResponse) error {
	if len(resp.Actions) == 0 {
		return fmt.Errorf("agent responded with empty action slice")
	}
	for _, action := range resp.Actions {
		switch action.Type {
		case "browser":
			ce.DispatchBrowserCommand(action)
		case "agent":
			ce.DispatchAgentCommand(action)
		default:
			return fmt.Errorf("unknown action type: %s", action.Type)
		}
	}
	return nil
}

// Browser command dispatch
func (ce *CommandEngine) DispatchBrowserCommand(action Action) {
	switch action.Name {

	case "navigate":
		var params NavigateParams
		if err := json.Unmarshal(action.Params, &params); err != nil {
			ce.Browser.Logger.Info(fmt.Sprintf("error unmarshalling navigate action: %v", err))
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
			ce.Browser.Logger.Info(fmt.Sprintf("error unmarshalling click action: %v", err))
		}

		// Description
		description := fmt.Sprintf("clicked node [%s]", params.NodeID)
		if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
			description = fmt.Sprintf("clicked node %s%s %q <%s> [%s]\n",
				browser.GetNodeRole(node), browser.GetNodeName(node),
				browser.GetNodeValue(node), params.NodeID)
		}

		// Execute action
		var result string
		if err := ce.Browser.Execute(ce.Browser.WaitReady("body"), ce.Browser.ClickNode(params.NodeID)); err != nil {
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
			ce.Browser.Logger.Info(fmt.Sprintf("error unmarshalling click action: %v", err))
		}

		// Description
		description := fmt.Sprintf("send keys %q to node [%s]", params.Keys, params.NodeID)
		if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
			description = fmt.Sprintf("sent keys %q node %s%s %q <%s> [%s]\n",
				params.Keys,
				browser.GetNodeRole(node), browser.GetNodeName(node),
				browser.GetNodeValue(node), params.NodeID)
		}

		// Execute action
		var result string
		if err := ce.Browser.Execute(ce.Browser.WaitReady("body"), ce.Browser.SendKeysNode(params.NodeID, params.Keys, params.Simulate)); err != nil {
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
		// TODO?
	}
}

// Agent command dispatch
func (ce *CommandEngine) DispatchAgentCommand(action Action) {
	switch action.Name {
	case "done":
		ce.ActionHistory = append(ce.ActionHistory, Action{
			Type: "agent",
			Name: "done",
		})

	default:
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
