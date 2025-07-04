package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"profiler/env"

	"google.golang.org/genai"
)

type (
	NewCategoryMessage struct{}

	AiMessage struct {
		Content string
	}

	AiThinkingMessage struct {
		Thinking bool
	}
)

type AiMessageHistoryEntry struct {
	Content string
	IsUser  bool
}

const modelName = "gemini-2.5-flash"

var (
	llmClient        *genai.Client
	aiMessageHistory []AiMessageHistoryEntry
)

func init() {
	ctx := context.Background()
	var err error
	llmClient, err = genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  env.GOOGLE_API_KEY,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating client: %v\n", err)
		os.Exit(1)
		return
	}
}

var (
	prompt = `You talk to a software developer about various specialist areas to find out their level in these specialist areas. Ask specific questions on the respective topic to determine the developer's experience. Don't talk about things that are opinions and where there is no objective right and wrong, these are irrelevant.
The current topic is: %s
If you were able to form an opinion about the developer, give a rating from 0 to 100. As long as you are still unsure, keep asking questions. Do not ask any more questions once you have a rating.`
	config = &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			AnyOf: []*genai.Schema{
				{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"message": {
							Type:     genai.TypeString,
							Nullable: genai.Ptr(false),
						},
					},
					Required: []string{"message"},
				},
				{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"rating": {
							Type:     genai.TypeInteger,
							Nullable: genai.Ptr(false),
							Minimum:  genai.Ptr(0.0),
							Maximum:  genai.Ptr(100.0),
						},
					},
					Required: []string{"rating"},
				},
			},
		},
	}
)

type AiResponse struct {
	Message *string `json:"message"`
	Rating  *int    `json:"rating"`
}

func Begin() {
	aiMessageHistory = nil
	teaProgram.Send(NewCategoryMessage{})
	teaProgram.Send(AiThinkingMessage{Thinking: true})
	ctx := context.Background()

	config.SystemInstruction = genai.NewContentFromText(fmt.Sprintf(prompt, GetCurrentCategory()), "model")

	resp, err := llmClient.Models.GenerateContent(
		ctx,
		modelName,
		genai.Text("Ask your first question"),
		config,
	)
	if err != nil {
		Err(errors.Join(errors.New("failed to generate content"), err))
		return
	}

	aiResp := AiResponse{}
	err = json.Unmarshal([]byte(resp.Text()), &aiResp)
	if err != nil {
		Err(errors.Join(errors.New("failed to unmarshal response"), err))
		return
	}

	teaProgram.Send(AiThinkingMessage{Thinking: false})

	if aiResp.Message != nil {
		teaProgram.Send(AiMessage{Content: *aiResp.Message})
	}

	if aiResp.Rating != nil {
		ApplyRating(*aiResp.Rating)
	}
}

func Continue(userInput string) {
	ctx := context.Background()
	teaProgram.Send(AiThinkingMessage{Thinking: true})

	config.SystemInstruction = genai.NewContentFromText(fmt.Sprintf(prompt, GetCurrentCategory()), "model")

	history := make([]*genai.Content, 0, len(aiMessageHistory)+1)
	for _, entry := range aiMessageHistory {
		if entry.IsUser {
			history = append(history, genai.NewContentFromText(entry.Content, "user"))
		} else {
			history = append(history, genai.NewContentFromText(entry.Content, "model"))
		}
	}
	history = append(history, genai.NewContentFromText(userInput, "user"))

	resp, err := llmClient.Models.GenerateContent(
		ctx,
		modelName,
		history,
		config,
	)
	if err != nil {
		Err(errors.Join(errors.New("failed to generate content"), err))
		return
	}

	aiResp := AiResponse{}
	err = json.Unmarshal([]byte(resp.Text()), &aiResp)
	if err != nil {
		Err(errors.Join(errors.New("failed to unmarshal response"), err))
		return
	}

	teaProgram.Send(AiThinkingMessage{Thinking: false})

	if aiResp.Message != nil {
		teaProgram.Send(AiMessage{Content: *aiResp.Message})
	}

	if aiResp.Rating != nil {
		ApplyRating(*aiResp.Rating)
	}

	return
}
