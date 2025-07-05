package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"profiler/env"
	"strings"
	"text/template"

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
	prompt = template.Must(
		template.New("prompt").
			Parse(`You are interviewing a software developer to assess their expertise in {{.Topic}}.
You will NOT use any generic greetings or chit-chat unrelated to {{.Topic}}.

1. WARM-UP (Topic-Focused)
   • Ask 1 open-ended question about a past project or role specifically
     involving {{.Topic}}.
     – Example: “Tell me about the most challenging {{.Topic}} project you’ve led.”
   • One question at a time. Never bundle or repeat questions.
   • If the candidate describes a project that involves {{.Topic}}:
     – Say once: “Acknowledged: {{.Topic}} project noted.”
     – Do NOT repeat that acknowledgement again.
   • After that single acknowledgement, immediately transition to DIVE-IN.

2. DIVE-IN
   • Ask a targeted technical question on {{.Topic}} about the project they
     just described.
   • If the candidate answers “I don’t know,” “I have no idea,” or indicates
     zero knowledge:
     – Say “Understood. Moving to evaluation.”  
     – Jump to step 3.
   • If their answer is incorrect or incomplete:
     – Give one-line feedback (e.g. “That’s not quite accurate; can you clarify X?”).  
     – Ask a different follow-up on the same subtopic.
   • After one follow-up without adequate progress, move to a new subtopic.
   • Limit to 3 total technical questions (excluding the “I don’t know” exit).

3. RATING & FEEDBACK
   • Stop asking questions and assign an integer rating from 1–100:
     – 100: Legend/mastery  
     – 75: Highly proficient  
     – 50: Competent mid-level  
     – 25: Foundational knowledge with gaps  
     – 10: Junior/basic theoretical  
     – 1: No understanding/misconceptions  
   • Do NOT change the initial baseline rating of 1—only increase it as you
     gather evidence.
   • Provide 1–2 sentences of actionable feedback on how to improve.

4. CLARIFICATIONS
   • If you don’t understand their wording, ask ONE short clarification, still
     on {{.Topic}}.
   • Never introduce off-topic pleasantries or multiple questions at once.

Current topic: {{.Topic}}  
Begin immediately with your first warm-up question.`))

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
							Minimum:  genai.Ptr(1.0),
							Maximum:  genai.Ptr(100.0),
						},
						"comment": {
							Type:     genai.TypeString,
							Nullable: genai.Ptr(false),
						},
					},
					Required: []string{"rating", "comment"},
				},
			},
		},
	}
)

type AiResponse struct {
	Message *string `json:"message"`
	Rating  *int    `json:"rating"`
	Comment *string `json:"comment"`
}

func getPromptString() string {
	sb := strings.Builder{}
	err := prompt.Execute(&sb, map[string]string{"Topic": GetCurrentCategory()})
	if err != nil {
		Err(errors.Join(errors.New("failed to execute prompt template"), err))
		return ""
	}
	return sb.String()
}

func Begin() {
	if GetCurrentCategoryScore() > 0 && !RedoTakenTests {
		ApplyRating(GetCurrentCategoryScore())
		return
	}
	aiMessageHistory = nil
	teaProgram.Send(NewCategoryMessage{})
	teaProgram.Send(AiThinkingMessage{Thinking: true})
	ctx := context.Background()

	config.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{{Text: getPromptString()}},
	}

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

	if aiResp.Comment != nil {
		ApplyComment(*aiResp.Comment)
	}

	if aiResp.Rating != nil {
		ApplyRating(*aiResp.Rating)
	}
}

func Continue(userInput string) {
	ctx := context.Background()
	teaProgram.Send(AiThinkingMessage{Thinking: true})

	config.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{{Text: getPromptString()}},
	}

	aiMessageHistory = append(aiMessageHistory, AiMessageHistoryEntry{
		Content: userInput,
		IsUser:  true,
	})
	history := make([]*genai.Content, 0, len(aiMessageHistory))
	for _, entry := range aiMessageHistory {
		if entry.Content == "" {
			Err(fmt.Errorf("empty content at index %d", len(aiMessageHistory)-1))
			return
		}
		if entry.IsUser {
			history = append(history, genai.NewContentFromText(entry.Content, "user"))
		} else {
			history = append(history, genai.NewContentFromText(entry.Content, "model"))
		}
	}

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

	if aiResp.Comment != nil {
		ApplyComment(*aiResp.Comment)
	}

	if aiResp.Rating != nil {
		ApplyRating(*aiResp.Rating)
	}
}
