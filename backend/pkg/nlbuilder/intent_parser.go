package nlbuilder

import (
	"regexp"
	"strings"
)

// BuiltinIntentParser provides a rule-based intent parser that works
// without an external LLM API. It handles common patterns and falls
// back gracefully for ambiguous inputs.
type BuiltinIntentParser struct{}

func (p *BuiltinIntentParser) Complete(systemPrompt string, messages []ConversationMessage) (string, error) {
	if len(messages) == 0 {
		return "How can I help you set up a webhook?", nil
	}
	last := messages[len(messages)-1]
	intent, _ := p.ParseIntent(last.Content)
	if intent.Confidence < 0.5 {
		return "Could you provide more details? For example: 'Send order.created events to https://api.example.com/webhooks'", nil
	}
	return "I've parsed your request. Here's what I understand.", nil
}

func (p *BuiltinIntentParser) ParseIntent(userMessage string) (*ParsedIntent, error) {
	msg := strings.ToLower(userMessage)
	intent := &ParsedIntent{
		RawQuery:   userMessage,
		Confidence: 0.3,
	}

	// Extract URL
	urlPattern := regexp.MustCompile(`https?://[^\s"']+`)
	if urls := urlPattern.FindAllString(userMessage, -1); len(urls) > 0 {
		intent.TargetURL = urls[0]
		intent.Confidence += 0.3
	}

	// Extract event types (patterns like "order.created", "payment.failed")
	eventPattern := regexp.MustCompile(`\b([a-z]+\.[a-z]+(?:\.[a-z]+)?)\b`)
	if events := eventPattern.FindAllString(msg, -1); len(events) > 0 {
		// Filter out common false positives
		var filtered []string
		for _, e := range events {
			if !isCommonPhrase(e) {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) > 0 {
			intent.EventTypes = filtered
			intent.Confidence += 0.2
		}
	}

	// Determine action
	if strings.Contains(msg, "retry") || strings.Contains(msg, "retries") || strings.Contains(msg, "backoff") {
		intent.Action = "configure_retry"
		intent.RetryPolicy = parseRetryPolicy(msg)
		intent.Confidence += 0.2
	} else if strings.Contains(msg, "transform") || strings.Contains(msg, "map") || strings.Contains(msg, "convert") {
		intent.Action = "add_transform"
		intent.Transform = parseTransform(msg)
		intent.Confidence += 0.2
	} else if strings.Contains(msg, "filter") || strings.Contains(msg, "only when") || strings.Contains(msg, "exclude") {
		intent.Action = "set_filter"
		intent.Filter = parseFilter(msg)
		intent.Confidence += 0.2
	} else {
		intent.Action = "create_endpoint"
	}

	// Cap confidence at 1.0
	if intent.Confidence > 1.0 {
		intent.Confidence = 1.0
	}

	return intent, nil
}

func parseRetryPolicy(msg string) *RetryPolicySpec {
	policy := &RetryPolicySpec{
		MaxRetries:  5,
		Strategy:    "exponential",
		InitialWait: "1s",
		MaxWait:     "1h",
	}

	if strings.Contains(msg, "linear") {
		policy.Strategy = "linear"
	} else if strings.Contains(msg, "fixed") {
		policy.Strategy = "fixed"
	}

	// Extract retry count
	retryPattern := regexp.MustCompile(`(\d+)\s*(?:retries|attempts|times)`)
	if matches := retryPattern.FindStringSubmatch(msg); len(matches) > 1 {
		count := 0
		for _, c := range matches[1] {
			count = count*10 + int(c-'0')
		}
		if count > 0 && count <= 20 {
			policy.MaxRetries = count
		}
	}

	return policy
}

func parseTransform(msg string) *TransformSpec {
	return &TransformSpec{
		Language:   "javascript",
		Expression: "// Auto-generated transform\nfunction transform(payload) {\n  return payload;\n}",
	}
}

func parseFilter(msg string) *FilterSpec {
	return &FilterSpec{
		Conditions: []FilterCondition{
			{
				Field:    "event_type",
				Operator: "eq",
				Value:    "*",
			},
		},
		Logic: "and",
	}
}

func isCommonPhrase(s string) bool {
	common := map[string]bool{
		"e.g": true, "i.e": true, "etc.com": true,
	}
	return common[s]
}
