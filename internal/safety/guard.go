// Package safety provides content safety filtering using Llama Guard 3.
package safety

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mabulgu/pawdy/pkg/types"
)

// Guard implements safety filtering using Llama Guard 3.
type Guard struct {
	client  types.LLMClient
	enabled bool
}

// NewGuard creates a new safety guard instance.
func NewGuard(client types.LLMClient, enabled bool) *Guard {
	return &Guard{
		client:  client,
		enabled: enabled,
	}
}

// CheckInput validates user input for safety violations.
func (g *Guard) CheckInput(ctx context.Context, text string) (*types.SafetyResult, error) {
	if !g.enabled {
		return &types.SafetyResult{IsSafe: true}, nil
	}

	prompt := g.buildInputPrompt(text)
	response, err := g.client.Generate(ctx, prompt, types.GenerateOptions{
		Temperature: 0.0, // Use deterministic output for safety
		MaxTokens:   100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check input safety: %w", err)
	}

	return g.parseResponse(response), nil
}

// CheckOutput validates model output for safety violations.
func (g *Guard) CheckOutput(ctx context.Context, text string) (*types.SafetyResult, error) {
	if !g.enabled {
		return &types.SafetyResult{IsSafe: true}, nil
	}

	prompt := g.buildOutputPrompt(text)
	response, err := g.client.Generate(ctx, prompt, types.GenerateOptions{
		Temperature: 0.0, // Use deterministic output for safety
		MaxTokens:   100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check output safety: %w", err)
	}

	return g.parseResponse(response), nil
}

// IsEnabled returns whether safety filtering is currently enabled.
func (g *Guard) IsEnabled() bool {
	return g.enabled
}

// buildInputPrompt constructs the safety check prompt for user input.
func (g *Guard) buildInputPrompt(text string) string {
	return fmt.Sprintf(`<|begin_of_text|><|start_header_id|>system<|end_header_id|>

You are Llama Guard 3, a content policy classifier. Your task is to classify the following user request into one of the safety categories defined below.

<BEGIN UNSAFE CONTENT CATEGORIES>
S1: Violent Crimes. 
S2: Non-Violent Crimes.
S3: Sex Crimes.
S4: Child Exploitation.
S5: Defamation.
S6: Specialized Advice.
S7: Privacy.
S8: Intellectual Property.
S9: Indiscriminate Weapons.
S10: Hate.
S11: Self-Harm.
S12: Sexual Content.
S13: Elections.
S14: Code Interpreter Abuse.
<END UNSAFE CONTENT CATEGORIES>

<BEGIN CONVERSATION>

<|eot_id|><|start_header_id|>user<|end_header_id|>

%s<|eot_id|><|start_header_id|>assistant<|end_header_id|>

`, text)
}

// buildOutputPrompt constructs the safety check prompt for model output.
func (g *Guard) buildOutputPrompt(text string) string {
	return fmt.Sprintf(`<|begin_of_text|><|start_header_id|>system<|end_header_id|>

You are Llama Guard 3, a content policy classifier. Your task is to classify the following AI assistant response into one of the safety categories defined below.

<BEGIN UNSAFE CONTENT CATEGORIES>
S1: Violent Crimes. 
S2: Non-Violent Crimes.
S3: Sex Crimes.
S4: Child Exploitation.
S5: Defamation.
S6: Specialized Advice.
S7: Privacy.
S8: Intellectual Property.
S9: Indiscriminate Weapons.
S10: Hate.
S11: Self-Harm.
S12: Sexual Content.
S13: Elections.
S14: Code Interpreter Abuse.
<END UNSAFE CONTENT CATEGORIES>

<BEGIN CONVERSATION>

<|eot_id|><|start_header_id|>assistant<|end_header_id|>

%s<|eot_id|><|start_header_id|>user<|end_header_id|>

Please classify this response.<|eot_id|><|start_header_id|>assistant<|end_header_id|>

`, text)
}

// parseResponse parses the Llama Guard response to determine safety.
func (g *Guard) parseResponse(response string) *types.SafetyResult {
	response = strings.TrimSpace(response)
	
	// Check for safe response
	if strings.ToLower(response) == "safe" {
		return &types.SafetyResult{
			IsSafe: true,
		}
	}

	// Check for unsafe response with category
	unsafePattern := regexp.MustCompile(`(?i)unsafe\s*(s\d+)?`)
	matches := unsafePattern.FindStringSubmatch(response)
	
	if len(matches) > 0 {
		category := ""
		reason := ""
		
		if len(matches) > 1 && matches[1] != "" {
			categoryCode := strings.ToUpper(matches[1])
			category = categoryCode
			if description, exists := types.SafetyCategories[categoryCode]; exists {
				reason = description
			}
		}

		return &types.SafetyResult{
			IsSafe:   false,
			Category: category,
			Reason:   reason,
		}
	}

	// Default to unsafe if we can't parse the response
	return &types.SafetyResult{
		IsSafe: false,
		Reason: "Unable to determine safety classification",
	}
}

// GetRefusalMessage returns an appropriate refusal message for unsafe content.
func GetRefusalMessage(category string) string {
	baseMessage := "I can't provide assistance with that request as it may violate content safety guidelines"
	
	if category == "" {
		return baseMessage + "."
	}
	
	categoryDescription, exists := types.SafetyCategories[category]
	if !exists {
		return baseMessage + "."
	}
	
	return fmt.Sprintf("%s (category: %s - %s).", baseMessage, category, categoryDescription)
}
