# json.query Processor for Conduit

A flexible JSON querying and transformation processor for [Conduit](https://conduit.io) that supports both JMESPath and jq syntax.

## Overview

The `json.query` processor enables powerful filtering and transformation of JSON payloads within Conduit pipelines. It evaluates query expressions against each message's JSON payload and replaces the payload with the query result. This processor is ideal for:

- Extracting specific fields from complex JSON structures
- Reshaping JSON data to match downstream requirements
- Filtering and aggregating JSON arrays
- Computing derived values from existing fields

## Quickstart

1. Build the processor:
```bash
make build
```

2. Create a pipeline configuration (`pipeline.yaml`):
```yaml
version: 2.2
pipelines:
  - id: example-pipeline
    connectors:
      - id: source
        type: generator
        plugin: builtin:generator
        settings:
          format.type: structured
          format.options.id: int
          format.options.name: string
          format.options.price: float
          
      - id: destination
        type: log
        plugin: builtin:log
        
    processors:
      - id: extract-name
        plugin: processor-jsonquery
        settings:
          type: jmespath
          query: name
```

3. Run the pipeline:
```bash
conduit -p pipeline.yaml
```

## Configuration Example

### Using JMESPath

```toml
[processor.json.query]
type = "jmespath"
query = "user.profile.email"
```

### Using jq

```toml
[processor.json.query]
type = "jq"
query = ".items | map(select(.active)) | length"
```

## Usage Examples

### Example 1: Extract Nested Fields (JMESPath)

**Input:**
```json
{
  "order": {
    "id": "ORD-123",
    "customer": {
      "name": "Alice Smith",
      "email": "alice@example.com",
      "tier": "premium"
    },
    "items": [
      {"product": "Widget", "price": 29.99},
      {"product": "Gadget", "price": 49.99}
    ]
  }
}
```

**Configuration:**
```toml
type = "jmespath"
query = "order.customer.email"
```

**Output:**
```json
"alice@example.com"
```

### Example 2: Filter and Count (jq)

**Input:**
```json
{
  "users": [
    {"name": "Alice", "active": true, "role": "admin"},
    {"name": "Bob", "active": false, "role": "user"},
    {"name": "Charlie", "active": true, "role": "user"},
    {"name": "David", "active": true, "role": "admin"}
  ]
}
```

**Configuration:**
```toml
type = "jq"
query = ".users | map(select(.active and .role == \"admin\")) | length"
```

**Output:**
```json
2
```

### Example 3: Transform Structure (JMESPath)

**Input:**
```json
{
  "products": [
    {"id": 1, "name": "Laptop", "price": 999.99, "category": "Electronics"},
    {"id": 2, "name": "Desk", "price": 299.99, "category": "Furniture"},
    {"id": 3, "name": "Mouse", "price": 29.99, "category": "Electronics"}
  ]
}
```

**Configuration:**
```toml
type = "jmespath"
query = "products[?category=='Electronics'].{id: id, name: name, price: price}"
```

**Output:**
```json
[
  {"id": 1, "name": "Laptop", "price": 999.99},
  {"id": 3, "name": "Mouse", "price": 29.99}
]
```

### Example 4: Calculate Aggregates (jq)

**Input:**
```json
{
  "sales": [
    {"date": "2024-01-01", "amount": 1200, "region": "North"},
    {"date": "2024-01-02", "amount": 850, "region": "South"},
    {"date": "2024-01-03", "amount": 2100, "region": "North"},
    {"date": "2024-01-04", "amount": 1500, "region": "East"}
  ]
}
```

**Configuration:**
```toml
type = "jq"
query = """
{
  total: .sales | map(.amount) | add,
  average: (.sales | map(.amount) | add) / (.sales | length),
  by_region: .sales | group_by(.region) | map({region: .[0].region, total: map(.amount) | add})
}
"""
```

**Output:**
```json
{
  "total": 5650,
  "average": 1412.5,
  "by_region": [
    {"region": "East", "total": 1500},
    {"region": "North", "total": 3300},
    {"region": "South", "total": 850}
  ]
}
```

### Example 5: Complex Pipeline Processing

```yaml
version: 2.2
pipelines:
  - id: order-processing
    connectors:
      - id: kafka-source
        type: kafka
        plugin: builtin:kafka
        settings:
          servers: "localhost:9092"
          topics: "raw-orders"
          
      - id: postgres-dest
        type: postgres
        plugin: builtin:postgres
        settings:
          url: "postgres://user:pass@localhost/orders"
          table: "processed_orders"
          
    processors:
      # Extract order summary
      - id: extract-summary
        plugin: processor-jsonquery
        settings:
          type: jmespath
          query: |
            {
              order_id: order.id,
              customer_email: order.customer.email,
              total_amount: order.items[*].price | sum(@),
              item_count: order.items | length(@)
            }
            
      # Add processing metadata with jq
      - id: add-metadata
        plugin: processor-jsonquery
        settings:
          type: jq
          query: |
            . + {
              processed_at: now | strftime("%Y-%m-%dT%H:%M:%SZ"),
              status: (if .total_amount > 1000 then "high_value" else "standard" end)
            }
```

## Build Instructions

Build the processor plugin:
```bash
make build
```

This creates a `processor-jsonquery` binary that can be loaded by Conduit.

## Test Instructions

Run the test suite:
```bash
make test
```

Run tests with coverage:
```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Run Instructions

For local debugging:
```bash
make run
```

To use in a Conduit pipeline, reference the built plugin in your pipeline configuration.

## Roadmap

- **Performance Optimization**: Cache compiled queries for repeated use
- **Additional Query Engines**: Support for JSONPath, XPath for JSON
- **Schema Validation**: Validate output against JSON Schema
- **Error Handling Modes**: Configure behavior on query errors (skip, default value, fail)
- **Batch Processing**: Optimize for processing multiple records with the same query
- **Query Templates**: Support for parameterized queries with runtime values
- **Metrics**: Export query performance metrics

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure:
- All tests pass (`make test`)
- Code is formatted (`make fmt`)
- Linting passes (`make lint`)
- Documentation is updated

## License

Apache License 2.0

Copyright 2024 Conduit Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.