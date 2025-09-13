package agent

import (
	"encoding/json"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidateFunc validates data against a JSON schema (bytes) and returns error on failure.
type ValidateFunc func(schema []byte, data any) error

// JSONSchemaValidator is a ValidateFunc using jsonschema/v6.
func JSONSchemaValidator(schema []byte, data any) error {
	if len(schema) == 0 {
		return nil
	}
	c := jsonschema.NewCompiler()
	// anonymous in-memory schema from parsed JSON
	var doc any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return err
	}
	if err := c.AddResource("mem://schema.json", doc); err != nil {
		return err
	}
	sch, err := c.Compile("mem://schema.json")
	if err != nil {
		return err
	}
	// Marshal/unmarshal to generic for validation
	b, _ := json.Marshal(data)
	var v any
	_ = json.Unmarshal(b, &v)
	return sch.Validate(v)
}

// CompileJSONSchema compiles the provided JSON schema and returns error only if the schema is invalid.
// It does not validate any instance data.
func CompileJSONSchema(schema []byte) error {
	if len(schema) == 0 {
		return nil
	}
	c := jsonschema.NewCompiler()
	var doc any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return err
	}
	if err := c.AddResource("mem://schema.json", doc); err != nil {
		return err
	}
	_, err := c.Compile("mem://schema.json")
	return err
}
