package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseYAMLDSL parses a YAML pipeline definition into a CreatePipelineRequest.
// This enables users to define multi-stage delivery pipelines declaratively.
//
// Example YAML:
//
//	name: order-processing
//	description: Process and route order webhooks
//	version: "1.0"
//	stages:
//	  - id: validate-payload
//	    type: validate
//	    config:
//	      schema: { "type": "object", "required": ["order_id"] }
//	      strictness: strict
//	  - id: enrich-metadata
//	    type: enrich
//	    config:
//	      script: "payload.processed_at = new Date().toISOString()"
//	  - id: route-by-type
//	    type: route
//	    config:
//	      rules:
//	        - condition: "payload.type == 'premium'"
//	          endpoint_ids: ["ep-premium"]
//	          label: premium-orders
//	        - condition: "true"
//	          endpoint_ids: ["ep-default"]
//	          label: default
func ParseYAMLDSL(yamlData []byte) (*CreatePipelineRequest, error) {
	var dsl PipelineDSL
	if err := yaml.Unmarshal(yamlData, &dsl); err != nil {
		return nil, fmt.Errorf("invalid pipeline YAML: %w", err)
	}
	return convertDSLToRequest(&dsl)
}

// PipelineYAMLDefinition is the full YAML schema for pipeline definitions,
// supporting multi-stage composition with transform, filter, route, fan-out,
// deliver, enrich, validate, delay, and log steps.
type PipelineYAMLDefinition struct {
	Name        string                `yaml:"name"`
	Description string                `yaml:"description,omitempty"`
	Version     string                `yaml:"version,omitempty"`
	Triggers    []PipelineTrigger     `yaml:"triggers,omitempty"`
	Variables   map[string]string     `yaml:"variables,omitempty"`
	Stages      []StageYAMLDefinition `yaml:"stages"`
	ErrorPolicy *ErrorPolicyConfig    `yaml:"error_policy,omitempty"`
	Metadata    map[string]string     `yaml:"metadata,omitempty"`
}

// PipelineTrigger defines what initiates the pipeline
type PipelineTrigger struct {
	EventType string            `yaml:"event_type" json:"event_type"`
	Source    string            `yaml:"source,omitempty" json:"source,omitempty"`
	Filter    string            `yaml:"filter,omitempty" json:"filter,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// StageYAMLDefinition is a single stage in the YAML DSL
type StageYAMLDefinition struct {
	ID              string       `yaml:"id"`
	Name            string       `yaml:"name,omitempty"`
	Type            string       `yaml:"type"`
	ContinueOnError bool         `yaml:"continue_on_error,omitempty"`
	Timeout         int          `yaml:"timeout,omitempty"`
	Condition       string       `yaml:"condition,omitempty"`
	RetryPolicy     *RetryPolicy `yaml:"retry,omitempty"`
	Config          interface{}  `yaml:"config,omitempty"`
}

// ErrorPolicyConfig defines global error handling for the pipeline
type ErrorPolicyConfig struct {
	OnFailure      string `yaml:"on_failure" json:"on_failure"` // abort, continue, dlq
	MaxRetries     int    `yaml:"max_retries" json:"max_retries"`
	RetryDelay     string `yaml:"retry_delay" json:"retry_delay"`
	DLQEndpointID  string `yaml:"dlq_endpoint_id,omitempty" json:"dlq_endpoint_id,omitempty"`
	AlertOnFailure bool   `yaml:"alert_on_failure" json:"alert_on_failure"`
}

// ParseFullYAML parses a complete YAML pipeline definition with triggers and error policies
func ParseFullYAML(yamlData []byte) (*PipelineYAMLDefinition, error) {
	var def PipelineYAMLDefinition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return nil, fmt.Errorf("invalid pipeline YAML: %w", err)
	}

	if err := validateYAMLDefinition(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

func validateYAMLDefinition(def *PipelineYAMLDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if len(def.Stages) == 0 {
		return fmt.Errorf("pipeline must have at least one stage")
	}

	ids := make(map[string]bool)
	for i, stage := range def.Stages {
		if stage.Type == "" {
			return fmt.Errorf("stage %d: type is required", i)
		}
		stageType := StageType(strings.ToLower(stage.Type))
		if err := validateStageType(stageType); err != nil {
			return fmt.Errorf("stage %d: %w", i, err)
		}
		if stage.ID != "" {
			if ids[stage.ID] {
				return fmt.Errorf("stage %d: duplicate id '%s'", i, stage.ID)
			}
			ids[stage.ID] = true
		}
		if stage.Timeout < 0 {
			return fmt.Errorf("stage %d: timeout must be non-negative", i)
		}
	}

	if def.ErrorPolicy != nil {
		validPolicies := map[string]bool{"abort": true, "continue": true, "dlq": true}
		if def.ErrorPolicy.OnFailure != "" && !validPolicies[def.ErrorPolicy.OnFailure] {
			return fmt.Errorf("error_policy.on_failure must be one of: abort, continue, dlq")
		}
	}

	return nil
}

// ConvertYAMLToCreateRequest converts a parsed YAML definition to a CreatePipelineRequest
func ConvertYAMLToCreateRequest(def *PipelineYAMLDefinition) (*CreatePipelineRequest, error) {
	dsl := &PipelineDSL{
		Name:        def.Name,
		Description: def.Description,
		Version:     def.Version,
	}

	for _, s := range def.Stages {
		dsl.Stages = append(dsl.Stages, StageDSL{
			ID:              s.ID,
			Name:            s.Name,
			Type:            s.Type,
			ContinueOnError: s.ContinueOnError,
			Timeout:         s.Timeout,
			Condition:       s.Condition,
			Config:          s.Config,
		})
	}

	return convertDSLToRequest(dsl)
}

// ExportToYAML exports a Pipeline to its YAML DSL representation
func ExportToYAML(p *Pipeline) ([]byte, error) {
	def := PipelineYAMLDefinition{
		Name:        p.Name,
		Description: p.Description,
		Version:     fmt.Sprintf("%d", p.Version),
	}

	for _, stage := range p.Stages {
		var config interface{}
		if stage.Config != nil {
			if err := json.Unmarshal(stage.Config, &config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config for stage %q: %w", stage.ID, err)
			}
		}

		def.Stages = append(def.Stages, StageYAMLDefinition{
			ID:              stage.ID,
			Name:            stage.Name,
			Type:            string(stage.Type),
			ContinueOnError: stage.ContinueOnError,
			Timeout:         stage.Timeout,
			Condition:       stage.Condition,
			Config:          config,
		})
	}

	return yaml.Marshal(&def)
}

// ValidateYAML validates a YAML pipeline definition without converting it
func ValidateYAML(yamlData []byte) []string {
	var errors []string

	var def PipelineYAMLDefinition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return []string{fmt.Sprintf("YAML parse error: %s", err.Error())}
	}

	if def.Name == "" {
		errors = append(errors, "pipeline name is required")
	}
	if len(def.Stages) == 0 {
		errors = append(errors, "pipeline must have at least one stage")
	}

	ids := make(map[string]bool)
	for i, stage := range def.Stages {
		if stage.Type == "" {
			errors = append(errors, fmt.Sprintf("stage %d: type is required", i))
			continue
		}
		stageType := StageType(strings.ToLower(stage.Type))
		if err := validateStageType(stageType); err != nil {
			errors = append(errors, fmt.Sprintf("stage %d: %s", i, err.Error()))
		}
		if stage.ID != "" && ids[stage.ID] {
			errors = append(errors, fmt.Sprintf("stage %d: duplicate id '%s'", i, stage.ID))
		}
		if stage.ID != "" {
			ids[stage.ID] = true
		}
	}

	return errors
}
