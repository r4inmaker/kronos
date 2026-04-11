package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
)

type AgentRequest struct {
	SystemPrompt  string							`json:"system_prompt"`
	Task 					string							`json:"task"`
	ActionHistory []SummarizedAction	`json:"action_history"`
	BrowserState  string							`json:"browser_state"`
}

type AgentResponse struct {
	Reasoning string 		`json:"reasoning"`
	Actions 	[]Action  `json:"actions"`
}

type SummarizedAction struct {
	Action string		 	`json:"action"`
	Result string 		`json:"result"`
}

type Action struct {
	Type 		string					`json:"type"`
	Name		string					`json:"name"`
	Params 	json.RawMessage	`json:"params"`
}

type NavigateParams struct {
	URL			string `json:"url"`
}

type ClickParams struct {
	NodeID  int64 `json:"node_id"`
}

type SendKeysParams struct {
	NodeID  		int64 	`json:"node_id"`
	Keys  			string 	`json:"keys"`
	Simulate		bool 		`json:"simulate"`
}

type UpdateHistoryParams struct {
    Index  int              `json:"index"`  
    Action SummarizedAction `json:"action"`
}


func GenerateSchema[T any]() map[string]any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference: 						true,
	}
	var v T
	schema := reflector.Reflect(v)

	data, _ := json.Marshal(schema)
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}

// Utility Functions
func (r *AgentResponse) FormatResponse() string {
	if len(r.Actions) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Reasoning: %s\n", r.Reasoning))
	sb.WriteString("Actions:\n")
	for i, action := range r.Actions {
		switch action.Type {
		case "browser":
			switch action.Name {
			case "navigate":
				var p NavigateParams
				json.Unmarshal(action.Params, &p)
				sb.WriteString(fmt.Sprintf("  [%d] navigate → %s\n", i, p.URL))
			case "click":
				var p ClickParams
				json.Unmarshal(action.Params, &p)
				sb.WriteString(fmt.Sprintf("  [%d] click → node %d\n", i, p.NodeID))
			case "send_keys":
				var p SendKeysParams
				json.Unmarshal(action.Params, &p)
				sb.WriteString(fmt.Sprintf("  [%d] send_keys → %q to node %d\n", i, p.Keys, p.NodeID))
			default:
				sb.WriteString(fmt.Sprintf("  [%d] unknown browser action: %s\n", i, action.Name))
			}
		case "agent":
			switch action.Name {
			case "update_history":
				var p UpdateHistoryParams
				json.Unmarshal(action.Params, &p)
				sb.WriteString(fmt.Sprintf("  [%d] update_history → index=%d %s\n", i, p.Index, p.Action.Action))
			case "done":
				sb.WriteString(fmt.Sprintf("  [%d] done\n", i))
			default:
				sb.WriteString(fmt.Sprintf("  [%d] unknown agent action: %s\n", i, action.Name))
			}
		default:
			sb.WriteString(fmt.Sprintf("  [%d] unknown type: %s\n", i, action.Type))
		}
	}
	return sb.String()
}