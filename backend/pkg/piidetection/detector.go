package piidetection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// builtinPatterns maps PII categories to their detection regexes.
var builtinPatterns = map[string]*regexp.Regexp{
	CategoryEmail:      regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
	CategoryPhone:      regexp.MustCompile(`(\+?1[\s.-]?)?\(?\d{3}\)?[\s.-]?\d{3}[\s.-]?\d{4}`),
	CategorySSN:        regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	CategoryCreditCard: regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
	CategoryIPAddress:  regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
}

// Detector scans JSON payloads for PII based on compiled patterns.
type Detector struct {
	categories    map[string]*regexp.Regexp
	maskingAction string
}

// NewDetector creates a Detector from a policy's categories and custom patterns.
func NewDetector(policy *Policy) *Detector {
	d := &Detector{
		categories:    make(map[string]*regexp.Regexp),
		maskingAction: policy.MaskingAction,
	}

	for _, cat := range policy.Categories {
		if p, ok := builtinPatterns[cat]; ok {
			d.categories[cat] = p
		}
	}

	for _, cp := range policy.CustomPatterns {
		if r, err := regexp.Compile(cp.Pattern); err == nil {
			label := cp.Label
			if label == "" {
				label = CategoryCustom
			}
			d.categories[label] = r
		}
	}
	return d
}

// ScanAndMask scans a JSON payload, masks detected PII, and returns the masked
// payload along with detection metadata.
func (d *Detector) ScanAndMask(payload json.RawMessage) (json.RawMessage, []Detection, int, int) {
	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, nil, 0, 0
	}

	var detections []Detection
	fieldsScanned := 0
	fieldsMasked := 0

	masked := d.walkAndMask(data, "", &detections, &fieldsScanned, &fieldsMasked)

	out, err := json.Marshal(masked)
	if err != nil {
		return payload, detections, fieldsScanned, fieldsMasked
	}
	return out, detections, fieldsScanned, fieldsMasked
}

func (d *Detector) walkAndMask(v interface{}, path string, detections *[]Detection, scanned, masked *int) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, child := range val {
			childPath := k
			if path != "" {
				childPath = path + "." + k
			}
			result[k] = d.walkAndMask(child, childPath, detections, scanned, masked)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(val))
		for i, child := range val {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			result[i] = d.walkAndMask(child, childPath, detections, scanned, masked)
		}
		return result

	case string:
		*scanned++
		for cat, pattern := range d.categories {
			if pattern.MatchString(val) {
				*detections = append(*detections, Detection{
					FieldPath: path,
					Category:  cat,
					Masked:    true,
				})
				*masked++
				return d.applyMask(val, pattern)
			}
		}
		return val

	default:
		return val
	}
}

func (d *Detector) applyMask(value string, pattern *regexp.Regexp) string {
	switch d.maskingAction {
	case ActionRedact:
		return pattern.ReplaceAllString(value, "[REDACTED]")
	case ActionHash:
		return pattern.ReplaceAllStringFunc(value, func(match string) string {
			h := sha256.Sum256([]byte(match))
			return hex.EncodeToString(h[:8])
		})
	case ActionTokenize:
		return pattern.ReplaceAllStringFunc(value, func(match string) string {
			if len(match) <= 4 {
				return strings.Repeat("*", len(match))
			}
			return strings.Repeat("*", len(match)-4) + match[len(match)-4:]
		})
	default: // ActionMask
		return pattern.ReplaceAllStringFunc(value, func(match string) string {
			if len(match) <= 4 {
				return strings.Repeat("*", len(match))
			}
			return strings.Repeat("*", len(match)-4) + match[len(match)-4:]
		})
	}
}

func hashPayload(data json.RawMessage) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func elapsedMs(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}
