package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/r4inmaker/kronos/internal/browser"
	"github.com/r4inmaker/kronos/internal/logger"
)

type Agent struct {
	Name 						string
	Task 						string
	SysPrompt	 			string
	Client 					openai.Client
	CommandEngine 	*CommandEngine
	Schema 					map[string]any
	Logger  				*logger.Logger
	ctx    	 				context.Context
	wg 							*sync.WaitGroup
}

func NewAgent(ctx context.Context, name string, sysPrompt string, task string, logger *logger.Logger) *Agent {
	// LLM client
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	// Response schema
	schema := GenerateSchema[AgentResponse]()

	// Browser
	b := browser.NewBrowser(ctx, logger)

	return &Agent{
		Name: 	name,
		Task:	 	task,
		Client: client,
		CommandEngine: NewCommandEngine(b, sysPrompt, task),
		Schema: schema,
		Logger: logger,
		ctx: 		ctx,
		wg: 		&sync.WaitGroup{},		
	}
}

func (a *Agent) Request() (*AgentResponse, error) {
    stateMsg := fmt.Sprintf("[Browser State]\n%s\n\n[Action History]\n%s",
        a.CommandEngine.Browser.SprintTree(a.CommandEngine.Browser.FilterFunc),
        a.CommandEngine.FormatActionHistory(10),
    )

    messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(a.CommandEngine.Prompt),
        openai.UserMessage(stateMsg),
    }

    resp, err := a.Client.Chat.Completions.New(
        a.ctx,
        openai.ChatCompletionNewParams{
            Model:    "gpt-4o",
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
    for {
        if err := a.CommandEngine.Browser.BuildFilteredTree(); err != nil {
            log.Fatal(err)
        }
				a.Logger.Info(a.CommandEngine.Browser.SprintTree(a.CommandEngine.Browser.FilterFunc))

        resp, err := a.Request()
        if err != nil {
            log.Fatal(err)
        }
        a.Logger.Info(resp.FormatResponse())
				
	      if err = a.CommandEngine.DispatchCommands(resp); err != nil {
            log.Fatal(err)
        }
    }
}


