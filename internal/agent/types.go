package agent

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

type AgentRequest struct {
	SystemPrompt  string   `json:"system_prompt"`
	Task          string   `json:"task"`
	ActionHistory []Action `json:"action_history"`
	BrowserState  string   `json:"browser_state"`
}

type AgentResponse struct {
	Reasoning string   `json:"reasoning"`
	Actions   []Action `json:"actions"`
}

type SummarizedMilestone struct {
	Milestone string `json:"reasoning"`
	Result    string `json:"result"`
}

type Action struct {
	Name        string          `json:"name" jsonschema:"enum=navigate,enum=click,enum=send_keys,enum=refresh-state,enum=done,description=The command to execute"`
	Description string          `json:"description" jsonschema:"description=A brief description of what the action will do"`
	Reasoning   string          `json:"reasoning" jsonschema:"description=The agent's logic for choosing this specific action"`
	Params      json.RawMessage `json:"params"`
}

type NavigateParams struct {
	URL string `json:"url"`
}

type ClickParams struct {
	NodeID int64 `json:"node_id"`
}

type SendKeysParams struct {
	NodeID int64  `json:"node_id"`
	Keys   string `json:"keys"`
	//Simulate bool   `json:"simulate"`
}

type UpdateHistoryParams struct {
	Index  int    `json:"index"`
	Action Action `json:"action"`
}

func GenerateSchema[T any]() map[string]any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)

	data, _ := json.Marshal(schema)
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}
