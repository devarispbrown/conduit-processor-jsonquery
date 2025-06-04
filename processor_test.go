package jsonquery

import (
	"context"
	"testing"

	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-processor-sdk"
	"github.com/matryer/is"
)

func TestProcessor_Configure(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
	}{
		{
			name: "valid jmespath config",
			config: config.Config{
				"type":  "jmespath",
				"query": "name",
			},
			wantErr: false,
		},
		{
			name: "valid jq config",
			config: config.Config{
				"type":  "jq",
				"query": ".name",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			config: config.Config{
				"type":  "invalid",
				"query": "name",
			},
			wantErr: true,
		},
		{
			name: "missing query",
			config: config.Config{
				"type": "jmespath",
			},
			wantErr: true,
		},
		{
			name: "invalid jmespath query",
			config: config.Config{
				"type":  "jmespath",
				"query": "[invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid jq query",
			config: config.Config{
				"type":  "jq",
				"query": ".[invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProcessor()
			err := p.Configure(ctx, tt.config)
			if tt.wantErr {
				is.True(err != nil)
			} else {
				is.NoErr(err)
			}
		})
	}
}

func TestProcessor_Process_JMESPath(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	p := NewProcessor()
	err := p.Configure(ctx, config.Config{
		"type":  "jmespath",
		"query": "user.name",
	})
	is.NoErr(err)
	is.NoErr(p.Open(ctx))

	// Test data
	input := opencdc.StructuredData{
		"user": map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		},
	}

	record := opencdc.Record{
		Position: []byte("test-pos-1"),
		Payload:  opencdc.Change{After: input},
	}

	results := p.Process(ctx, []opencdc.Record{record})
	is.Equal(len(results), 1)

	processed := results[0].(sdk.SingleRecord)
	// For scalar results, the processor returns RawData
	rawData := processed.Payload.After.(opencdc.RawData)
	is.Equal(string(rawData), `"John Doe"`)
}

func TestProcessor_Process_JQ(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	p := NewProcessor()
	err := p.Configure(ctx, config.Config{
		"type":  "jq",
		"query": ".items | map(.price) | add",
	})
	is.NoErr(err)
	is.NoErr(p.Open(ctx))

	// Test data
	input := opencdc.StructuredData{
		"items": []interface{}{
			map[string]interface{}{"name": "apple", "price": 1.5},
			map[string]interface{}{"name": "banana", "price": 0.75},
			map[string]interface{}{"name": "orange", "price": 2.0},
		},
	}

	record := opencdc.Record{
		Position: []byte("test-pos-2"),
		Payload:  opencdc.Change{After: input},
	}

	results := p.Process(ctx, []opencdc.Record{record})
	is.Equal(len(results), 1)

	processed := results[0].(sdk.SingleRecord)
	is.Equal(processed.Payload.After.(opencdc.StructuredData), 4.25)
}

func TestProcessor_Process_RawJSON(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	p := NewProcessor()
	err := p.Configure(ctx, config.Config{
		"type":  "jmespath",
		"query": "status",
	})
	is.NoErr(err)
	is.NoErr(p.Open(ctx))

	// Test with raw JSON payload
	jsonData := []byte(`{"status": "active", "count": 42}`)
	record := opencdc.Record{
		Position: []byte("test-pos-3"),
		Payload:  opencdc.Change{After: opencdc.RawData(jsonData)},
	}

	results := p.Process(ctx, []opencdc.Record{record})
	is.Equal(len(results), 1)

	processed := results[0].(sdk.SingleRecord)
	is.Equal(processed.Payload.After.(opencdc.StructuredData), "active")
}

func TestProcessor_Process_InvalidJSON(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	p := NewProcessor()
	err := p.Configure(ctx, config.Config{
		"type":  "jmespath",
		"query": "test",
	})
	is.NoErr(err)
	is.NoErr(p.Open(ctx))

	// Test with invalid JSON
	record := opencdc.Record{
		Position: []byte("test-pos-4"),
		Payload:  opencdc.Change{After: opencdc.RawData([]byte("invalid json"))},
	}

	results := p.Process(ctx, []opencdc.Record{record})
	is.Equal(len(results), 0) // Record should be skipped
}

func TestProcessor_Process_ComplexJQ(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	p := NewProcessor()
	err := p.Configure(ctx, config.Config{
		"type":  "jq",
		"query": `{name: .user.name, total: .orders | map(.amount) | add}`,
	})
	is.NoErr(err)
	is.NoErr(p.Open(ctx))

	input := opencdc.StructuredData{
		"user": map[string]interface{}{
			"name": "Alice",
			"id":   123,
		},
		"orders": []interface{}{
			map[string]interface{}{"id": 1, "amount": 100.0},
			map[string]interface{}{"id": 2, "amount": 50.5},
			map[string]interface{}{"id": 3, "amount": 75.25},
		},
	}

	record := opencdc.Record{
		Position: []byte("test-pos-5"),
		Payload:  opencdc.Change{After: input},
	}

	results := p.Process(ctx, []opencdc.Record{record})
	is.Equal(len(results), 1)

	processed := results[0].(sdk.SingleRecord)
	result := processed.Payload.After.(opencdc.StructuredData)
	is.Equal(result["name"], "Alice")
	is.Equal(result["total"], 225.75)
}
