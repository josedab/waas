package mocking

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	mathrand "math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Generator generates mock data
type Generator struct {
	rng *mathrand.Rand
}

// NewGenerator creates a new mock data generator
func NewGenerator() *Generator {
	return &Generator{
		rng: mathrand.New(mathrand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
}

// GeneratePayload generates a mock payload from a template
func (g *Generator) GeneratePayload(template *PayloadTemplate) (map[string]interface{}, error) {
	if template == nil {
		return map[string]interface{}{}, nil
	}

	// Start with static content
	result := make(map[string]interface{})
	for k, v := range template.Content {
		result[k] = v
	}

	// Apply dynamic fields
	for _, field := range template.Fields {
		value, err := g.generateFieldValue(field)
		if err != nil {
			return nil, fmt.Errorf("failed to generate field %s: %w", field.Path, err)
		}
		setNestedValue(result, field.Path, value)
	}

	return result, nil
}

// generateFieldValue generates a value for a template field
func (g *Generator) generateFieldValue(field TemplateField) (interface{}, error) {
	// Check for nullable
	if field.Options.Nullable && g.rng.Float64() < field.Options.NullProb {
		return nil, nil
	}

	// If static value provided
	if field.Value != nil && field.Faker == "" {
		return field.Value, nil
	}

	// Generate based on faker type
	return g.generateFakerValue(FakerType(field.Faker), field.Options)
}

// generateFakerValue generates a fake value based on type
func (g *Generator) generateFakerValue(fakerType FakerType, opts FieldOptions) (interface{}, error) {
	switch fakerType {
	case FakerUUID:
		return uuid.New().String(), nil
	case FakerEmail:
		return fmt.Sprintf("%s@example.com", g.randomString(10)), nil
	case FakerName:
		return g.randomName(), nil
	case FakerFirstName:
		return g.randomFirstName(), nil
	case FakerLastName:
		return g.randomLastName(), nil
	case FakerPhone:
		return g.randomPhone(), nil
	case FakerAddress:
		return g.randomAddress(), nil
	case FakerCity:
		return g.randomCity(), nil
	case FakerCountry:
		return g.randomCountry(), nil
	case FakerCompany:
		return g.randomCompany(), nil
	case FakerURL:
		return fmt.Sprintf("https://%s.example.com", g.randomString(8)), nil
	case FakerIPv4:
		return g.randomIPv4(), nil
	case FakerIPv6:
		return g.randomIPv6(), nil
	case FakerTimestamp:
		return time.Now().Format(time.RFC3339), nil
	case FakerDate:
		format := opts.Format
		if format == "" {
			format = "2006-01-02"
		}
		return time.Now().Format(format), nil
	case FakerNumber:
		min := int64(opts.Min)
		max := int64(opts.Max)
		if max == 0 {
			max = 1000
		}
		return min + g.rng.Int64N(max-min+1), nil
	case FakerFloat:
		min := opts.Min
		max := opts.Max
		if max == 0 {
			max = 1000
		}
		return min + g.rng.Float64()*(max-min), nil
	case FakerBoolean:
		return g.rng.Float64() > 0.5, nil
	case FakerWord:
		return g.randomWord(), nil
	case FakerSentence:
		return g.randomSentence(), nil
	case FakerParagraph:
		return g.randomParagraph(), nil
	case FakerCreditCard:
		return g.randomCreditCard(), nil
	case FakerCurrency:
		return g.randomCurrency(), nil
	case FakerPrice:
		min := opts.Min
		max := opts.Max
		if max == 0 {
			max = 1000
		}
		price := min + g.rng.Float64()*(max-min)
		return math.Round(price*100) / 100, nil
	case FakerUsername:
		return fmt.Sprintf("user_%s", g.randomString(8)), nil
	case FakerPassword:
		length := opts.Length
		if length == 0 {
			length = 16
		}
		return g.randomPassword(length), nil
	case FakerSlug:
		return strings.ToLower(fmt.Sprintf("%s-%s-%s", g.randomWord(), g.randomWord(), g.randomString(4))), nil
	case FakerHexColor:
		return fmt.Sprintf("#%s", g.randomHex(6)), nil
	case FakerUserAgent:
		return g.randomUserAgent(), nil
	default:
		// If choices provided, pick from them
		if len(opts.Choices) > 0 {
			return opts.Choices[g.rng.IntN(len(opts.Choices))], nil
		}
		return g.randomString(10), nil
	}
}

// Helper functions

func (g *Generator) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[g.rng.IntN(len(charset))]
	}
	return string(result)
}

func (g *Generator) randomHex(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (g *Generator) randomFirstName() string {
	names := []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "David", "Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica"}
	return names[g.rng.IntN(len(names))]
}

func (g *Generator) randomLastName() string {
	names := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas"}
	return names[g.rng.IntN(len(names))]
}

func (g *Generator) randomName() string {
	return fmt.Sprintf("%s %s", g.randomFirstName(), g.randomLastName())
}

func (g *Generator) randomPhone() string {
	return fmt.Sprintf("+1%d%d%d%d%d%d%d%d%d%d",
		g.rng.IntN(10), g.rng.IntN(10), g.rng.IntN(10),
		g.rng.IntN(10), g.rng.IntN(10), g.rng.IntN(10),
		g.rng.IntN(10), g.rng.IntN(10), g.rng.IntN(10), g.rng.IntN(10))
}

func (g *Generator) randomAddress() string {
	return fmt.Sprintf("%d %s St", g.rng.IntN(9999)+1, g.randomWord())
}

func (g *Generator) randomCity() string {
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose", "Austin", "Jacksonville", "Fort Worth", "Columbus", "Charlotte", "Seattle"}
	return cities[g.rng.IntN(len(cities))]
}

func (g *Generator) randomCountry() string {
	countries := []string{"United States", "Canada", "United Kingdom", "Germany", "France", "Australia", "Japan", "Brazil", "India", "Mexico", "Spain", "Italy", "Netherlands", "Sweden", "Switzerland"}
	return countries[g.rng.IntN(len(countries))]
}

func (g *Generator) randomCompany() string {
	prefixes := []string{"Tech", "Global", "Dynamic", "Innovative", "Smart", "Cloud", "Digital", "Future", "Prime", "Next"}
	suffixes := []string{"Corp", "Inc", "LLC", "Solutions", "Systems", "Technologies", "Labs", "Group", "Services", "Ventures"}
	return fmt.Sprintf("%s %s", prefixes[g.rng.IntN(len(prefixes))], suffixes[g.rng.IntN(len(suffixes))])
}

func (g *Generator) randomIPv4() string {
	return fmt.Sprintf("%d.%d.%d.%d", g.rng.IntN(256), g.rng.IntN(256), g.rng.IntN(256), g.rng.IntN(256))
}

func (g *Generator) randomIPv6() string {
	parts := make([]string, 8)
	for i := range parts {
		parts[i] = fmt.Sprintf("%x", g.rng.IntN(65536))
	}
	return strings.Join(parts, ":")
}

func (g *Generator) randomWord() string {
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa", "lambda", "mu", "nu", "xi", "omicron", "pi", "rho", "sigma", "tau", "upsilon", "phi", "chi", "psi", "omega"}
	return words[g.rng.IntN(len(words))]
}

func (g *Generator) randomSentence() string {
	words := make([]string, g.rng.IntN(10)+5)
	for i := range words {
		words[i] = g.randomWord()
	}
	sentence := strings.Join(words, " ")
	return strings.ToUpper(sentence[:1]) + sentence[1:] + "."
}

func (g *Generator) randomParagraph() string {
	sentences := make([]string, g.rng.IntN(5)+3)
	for i := range sentences {
		sentences[i] = g.randomSentence()
	}
	return strings.Join(sentences, " ")
}

func (g *Generator) randomCreditCard() string {
	// Generates test card number (not real)
	prefix := "4" // Visa-like
	number := prefix
	for len(number) < 16 {
		number += fmt.Sprintf("%d", g.rng.IntN(10))
	}
	return number
}

func (g *Generator) randomCurrency() string {
	currencies := []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "CHF", "CNY", "INR", "BRL"}
	return currencies[g.rng.IntN(len(currencies))]
}

func (g *Generator) randomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[g.rng.IntN(len(charset))]
	}
	return string(result)
}

func (g *Generator) randomUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
	}
	return agents[g.rng.IntN(len(agents))]
}

// setNestedValue sets a value at a nested path (e.g., "user.profile.name")
func setNestedValue(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}

		if nested, ok := current[part].(map[string]interface{}); ok {
			current = nested
		} else {
			// Can't traverse further, overwrite
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
}
