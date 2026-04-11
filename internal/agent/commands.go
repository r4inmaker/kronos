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
		Prompt 				string
    ActionHistory []SummarizedAction
}

func NewCommandEngine(browser *browser.Browser, sysPrompt, task string) *CommandEngine {
	initPrompt := fmt.Sprintf("[Objective]\n%s\n\n[Task]\n%s\n", sysPrompt, task)

	return &CommandEngine{
		Browser: browser,
		Prompt: initPrompt,
		ActionHistory: make([]SummarizedAction, 0),
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
            if err := ce.DispatchBrowserCommand(action); err != nil {
                return err
            }
        case "agent":
            if err := ce.DispatchAgentCommand(action); err != nil {
                return err
            }
        default:
            return fmt.Errorf("unknown action type: %s", action.Type)
        }
    }
    return nil
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
		if err := ce.Browser.Execute(ce.Browser.Navigate(params.URL)); err != nil {
			ce.ActionHistory = append(ce.ActionHistory, SummarizedAction{
            Action: fmt.Sprintf("navigate to %s", params.URL),
            Result: fmt.Sprintf("failed: %s", err),
      })
			return err
		}
		// Append to action history
		ce.ActionHistory = append(ce.ActionHistory, SummarizedAction{
				Action: fmt.Sprintf("navigate to %s", params.URL),
				Result: "success",
		})

	case "click":
			var params ClickParams
			if err := json.Unmarshal(action.Params, &params); err != nil {
					return err
			}
			desc := fmt.Sprintf("click node [%d]", params.NodeID)
			if node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]; ok {
					desc = fmt.Sprintf("click %s %q [%d]", browser.GetNodeRole(node), browser.GetNodeName(node), params.NodeID)
			}
			if err := ce.Browser.Execute(ce.Browser.WaitReady("body"), ce.Browser.ClickNode(params.NodeID)); err != nil {
					ce.ActionHistory = append(ce.ActionHistory, SummarizedAction{
							Action: desc,
							Result: fmt.Sprintf("failed: %s", err),
					})
					return err
			}
			ce.ActionHistory = append(ce.ActionHistory, SummarizedAction{
					Action: desc,
					Result: "attempted",
			})

	case "send_keys":
			var params SendKeysParams
			if err := json.Unmarshal(action.Params, &params); err != nil {
					return err
			}
			node, ok := ce.Browser.NodeMap[nodeID(params.NodeID)]
			if !ok {
					return fmt.Errorf("node %d not found in tree", params.NodeID)
			}
			desc := fmt.Sprintf("send_keys %q to %s %q [%d]", params.Keys, browser.GetNodeRole(node), browser.GetNodeName(node), params.NodeID)
			if err := ce.Browser.Execute(ce.Browser.WaitReady("body"), ce.Browser.SendKeysNode(params.NodeID, params.Keys, params.Simulate)); err != nil {
					ce.ActionHistory = append(ce.ActionHistory, SummarizedAction{
							Action: desc,
							Result: fmt.Sprintf("failed: %s", err),
					})
					return err
			}
			ce.ActionHistory = append(ce.ActionHistory, SummarizedAction{
					Action: desc,
					Result: "attempted",
			})

	default: return fmt.Errorf("invalid command name")
	}
	return nil
}

// Agent command dispatch
func (ce *CommandEngine) DispatchAgentCommand(action Action) error {
    switch action.Name {

    case "update_history":
        var params UpdateHistoryParams
        if err := json.Unmarshal(action.Params, &params); err != nil {
            return err
        }
        if params.Index == -1 {
            ce.ActionHistory = append(ce.ActionHistory, params.Action)
        } else {
            if params.Index < 0 || params.Index >= len(ce.ActionHistory) {
                return fmt.Errorf("history index %d out of range", params.Index)
            }
            ce.ActionHistory[params.Index] = params.Action
        }
		case "request_history":

    default:
        return fmt.Errorf("invalid command name: %s", action.Name)
    }
    return nil
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
    for i, a := range history {
        sb.WriteString(fmt.Sprintf("[%d] %s → %s\n", i, a.Action, a.Result))
    }
    return sb.String()
}

func nodeID(id int64) accessibility.NodeID {
    return accessibility.NodeID(strconv.Itoa(int(id)))
}