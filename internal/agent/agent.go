package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/r4inmaker/kronos/internal/browser"
	"github.com/r4inmaker/kronos/internal/logger"
)

type Agent struct {
	Name          string
	Task          string
	SysPrompt     string
	Client        openai.Client
	Schema        map[string]any
	Logger        *logger.Logger
	ctx           context.Context
	cancelFunc    context.CancelFunc
	wg            *sync.WaitGroup
	*CommandEngine
}

func NewAgent(ctx context.Context, cancelFunc context.CancelFunc, name string, sysPrompt string, task string, logger *logger.Logger) *Agent {
  // Use the DeepSeek API key and BaseURL
  client := openai.NewClient(
    option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    option.WithBaseURL("https://api.openai.com/v1"),
  )

  schema := GenerateSchema[AgentResponse]()
  b := browser.NewBrowser(ctx, logger)

  return &Agent{
    Name:          name,
    Task:          task,
    Client:        client,
    CommandEngine: NewCommandEngine(b, sysPrompt, task, cancelFunc),
    Schema:        schema,
    Logger:        logger,
    ctx:           ctx,
    cancelFunc:    cancelFunc,
    wg:            &sync.WaitGroup{},
  }
}

func (a *Agent) Request() (*AgentResponse, error) {
	stateMsg := fmt.Sprintf("[Browser State]\n%s\n\n[Action History]\n%s",
		a.Browser.SprintTree(a.CommandEngine.Browser.FilterFunc),
		a.FormatActionHistory(10),
	)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(a.CommandEngine.Prompt),
		openai.UserMessage(stateMsg),
	}

	resp, err := a.Client.Chat.Completions.New(
		a.ctx,
		openai.ChatCompletionNewParams{
			Model:    "gpt-4o-mini",
			Messages: messages,
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
					JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:   "agent_response",
						Schema: a.Schema,
						Strict: openai.Bool(true),
					},
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	content := resp.Choices[0].Message.Content

	var agentResp AgentResponse
	if err := json.Unmarshal([]byte(content), &agentResp); err != nil {
		return nil, err
	}

	return &agentResp, nil
}



func (a *Agent) Run() {
	waitFlag := false

	for {
		// Wait until DOM is stabilized
		if waitFlag {
			a.Browser.Execute(
				 a.Browser.WaitForLifecycle("networkIdle", 5*time.Second),
			)
		}
		waitFlag = true

		if err := a.Browser.BuildTree(); err != nil {
			log.Fatal(err)
		}
		//a.Logger.Info(fmt.Sprintf("NodeTreeDim: %d", len(a.Browser.Nodes)))

		resp, err := a.Request()
		if err != nil {
			log.Fatal(err)
		}

		exit, err := a.DispatchCommands(resp)
		if err != nil {
			log.Fatal(err)
		}

		if exit {
			return
		}
	}
}
