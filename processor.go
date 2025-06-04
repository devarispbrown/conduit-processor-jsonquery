// Package jsonquery provides a Conduit processor for querying and transforming JSON payloads
// using either JMESPath or jq expressions.
package jsonquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-processor-sdk"
	"github.com/itchyny/gojq"
	"github.com/jmespath/go-jmespath"
)

// Processor implements the Conduit processor interface for JSON querying.
type Processor struct {
	// Configuration
	config ProcessorConfig

	// Query engines
	jmesCompiled *jmespath.JMESPath
	jqCompiled   *gojq.Query

	sdk.UnimplementedProcessor
}

// ProcessorConfig defines the configuration for the json.query processor.
type ProcessorConfig struct {
	// Type specifies which query engine to use: "jmespath" or "jq"
	Type string `json:"type" validate:"required,oneof=jmespath jq"`

	// Query contains the expression to evaluate against each JSON payload
	Query string `json:"query" validate:"required"`
}

// Specification returns the processor specification.
func Specification() sdk.Specification {
	return sdk.Specification{
		Name:        "json.query",
		Summary:     "Query and transform JSON payloads using JMESPath or jq expressions",
		Description: "This processor allows filtering and transformation of JSON messages using either JMESPath or jq syntax. It evaluates the specified query against each message's JSON payload and replaces the payload with the query result.",
		Version:     "v0.1.0",
		Author:      "Conduit Community",
		Parameters: map[string]config.Parameter{
			"type": {
				Description: "Query engine type: 'jmespath' or 'jq'",
				Type:        config.ParameterTypeString,
				Default:     "",
				Validations: []config.Validation{
					config.ValidationRequired{},
					config.ValidationInclusion{List: []string{"jmespath", "jq"}},
				},
			},
			"query": {
				Description: "Query expression to evaluate against JSON payloads",
				Type:        config.ParameterTypeString,
				Default:     "",
				Validations: []config.Validation{
					config.ValidationRequired{},
				},
			},
		},
	}
}

// NewProcessor creates a new json.query processor instance.
func NewProcessor() sdk.Processor {
	return &Processor{}
}

// Configure validates and stores the processor configuration.
func (p *Processor) Configure(ctx context.Context, cfg config.Config) error {
	sdk.Logger(ctx).Info().Msg("Configuring json.query processor")

	// Parse configuration
	err := sdk.ParseConfig(ctx, cfg, &p.config, Specification().Parameters)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate and compile query based on type
	switch strings.ToLower(p.config.Type) {
	case "jmespath":
		p.jmesCompiled, err = jmespath.Compile(p.config.Query)
		if err != nil {
			return fmt.Errorf("invalid JMESPath expression: %w", err)
		}
		sdk.Logger(ctx).Info().Str("type", "jmespath").Str("query", p.config.Query).Msg("Compiled JMESPath query")

	case "jq":
		query, err := gojq.Parse(p.config.Query)
		if err != nil {
			return fmt.Errorf("invalid jq expression: %w", err)
		}
		p.jqCompiled = query
		sdk.Logger(ctx).Info().Str("type", "jq").Str("query", p.config.Query).Msg("Compiled jq query")

	default:
		return fmt.Errorf("unsupported query type: %s", p.config.Type)
	}

	return nil
}

// Open initializes the processor.
func (p *Processor) Open(ctx context.Context) error {
	sdk.Logger(ctx).Info().Msg("json.query processor opened")
	return nil
}

// Process applies the configured query to each record's payload.
func (p *Processor) Process(ctx context.Context, records []opencdc.Record) []sdk.ProcessedRecord {
	logger := sdk.Logger(ctx)
	results := make([]sdk.ProcessedRecord, 0, len(records))

	for _, record := range records {
		processed, err := p.processRecord(ctx, record)
		if err != nil {
			logger.Error().Err(err).
				Str("position", string(record.Position)).
				Msg("Failed to process record")
			// Skip records that fail processing
			continue
		}
		results = append(results, processed)
	}

	return results
}

// processRecord processes a single record.
func (p *Processor) processRecord(ctx context.Context, record opencdc.Record) (sdk.ProcessedRecord, error) {
	// Extract payload data
	var data interface{}

	// Check if we have data in the After field (following OpenCDC convention)
	if record.Payload.After != nil {
		switch payload := record.Payload.After.(type) {
		case opencdc.StructuredData:
			// Convert StructuredData to a regular map for processing
			data = convertStructuredData(payload)
		case opencdc.RawData:
			// Parse raw JSON data
			if err := json.Unmarshal(payload, &data); err != nil {
				return nil, fmt.Errorf("invalid JSON payload: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported payload type: %T", payload)
		}
	} else {
		return nil, errors.New("no data in record.Payload.After")
	}

	// Apply query based on configured type
	var result interface{}
	var err error

	switch strings.ToLower(p.config.Type) {
	case "jmespath":
		result, err = p.jmesCompiled.Search(data)
		if err != nil {
			return nil, fmt.Errorf("JMESPath query failed: %w", err)
		}

	case "jq":
		iter := p.jqCompiled.Run(data)
		// Get first result from jq iterator
		val, ok := iter.Next()
		if !ok {
			return nil, errors.New("jq query produced no results")
		}
		if err, ok := val.(error); ok {
			return nil, fmt.Errorf("jq query failed: %w", err)
		}
		result = val
	}

	// Create a copy of the record and update its payload
	recordCopy := record

	// Set the result based on its type
	switch v := result.(type) {
	case map[string]interface{}:
		// Maps are stored as StructuredData
		recordCopy.Payload.After = opencdc.StructuredData(v)
	case []interface{}:
		// Arrays need to be wrapped in a map for StructuredData
		recordCopy.Payload.After = opencdc.StructuredData(map[string]interface{}{
			"result": v,
		})
	default:
		// For scalar values (string, number, bool, nil), convert to JSON and store as RawData
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		recordCopy.Payload.After = opencdc.RawData(jsonBytes)
	}

	sdk.Logger(ctx).Debug().
		Str("position", string(record.Position)).
		Str("query_type", p.config.Type).
		Interface("result", result).
		Msg("Successfully processed record")

	return sdk.SingleRecord(recordCopy), nil
}

// convertStructuredData recursively converts opencdc.StructuredData to regular maps/slices
func convertStructuredData(data interface{}) interface{} {
	switch v := data.(type) {
	case opencdc.StructuredData:
		// Convert to regular map
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertStructuredData(value)
		}
		return result
	case map[string]interface{}:
		// Recursively convert map values
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertStructuredData(value)
		}
		return result
	case []interface{}:
		// Recursively convert slice elements
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = convertStructuredData(value)
		}
		return result
	default:
		// Return as-is for primitive types
		return v
	}
}

// Teardown cleans up processor resources.
func (p *Processor) Teardown(ctx context.Context) error {
	sdk.Logger(ctx).Info().Msg("json.query processor teardown complete")
	return nil
}
