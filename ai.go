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
	prompt = `You talk to a software developer about various specialist areas to find out their level in these specialist areas. Ask specific questions on the respective topic to determine the developer's experience. Don't elaborate on things that are opinions and where there is no objective right and wrong, these are irrelevant.

Ask only one question at a time. Do not ask multiple questions at once. Do not ask the same question multiple times. If you are unsure about the answer, say so and ask for clarification.
Leave the questins open enough to allow the developer to talk about their experience outside of the specific question. But ask specific enough so that it is clear what you are asking for.

Make sure that the answer is not in the question.

Don't ask more than three questions about the same topic.

The current topic is: %s

If you were able to form an opinion about the developer, give a rating from 1 to 100. This is a continuous scale, and you should use any integer between 0 and 100 that best reflects your assessment. As long as you are still unsure, keep asking questions. Do not ask any more questions once you have a rating.

Generally, someone who tries to prevent complexity is considered a wise developer. Someone who shies away from complexity is likely a beginner.

Use the following benchmarks as guidance, but feel free to select any score within the range that accurately represents the developer's knowledge:

*   100: This score is reserved for developers who demonstrate an excellent, profound understanding of the topic, comparable to a recognized legend in the field (e.g., Dan Abramov for React, Ken Thompson for compilers, John Carmack for Game Engines). While no one knows absolutely everything, a 100 indicates a deep grasp of core principles and advanced intricacies. Do not hesitate to give a 100 if the developer truly exhibits this level of mastery.
*   75: A developer scoring around 75 would be considered highly proficient, demonstrating strong problem-solving skills and a very solid understanding of both common and more complex aspects of the field, exceeding the typical mid-level expectations.
*   50: This score represents the expected level of a competent mid-range developer currently working in the field. They possess a good working knowledge of the topic, can solve most common problems independently, and understand standard practices.
*   25: A score around 25 indicates a developer who has some foundational knowledge, perhaps beyond a complete novice but still with significant gaps or limited practical experience. They might be able to handle basic tasks with some guidance.
*   10: This score signifies a junior developer, likely someone right out of college or with minimal practical experience in this specific field, who has a basic theoretical understanding but lacks depth or practical application.
*   1: If the developer demonstrates virtually no understanding or has significant misconceptions about the topic, a score of 1 is appropriate.

Also give a comment on why you gave the score you did. This will help the developer improve their skills and knowledge. If you don't give 100 as a rating, you must give a comment that explains why. If you can't find anything negative, you have to give a rating of 100.`
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
		Parts: []*genai.Part{{Text: fmt.Sprintf(prompt, GetCurrentCategory())}},
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
		Parts: []*genai.Part{{Text: fmt.Sprintf(prompt, GetCurrentCategory())}},
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
