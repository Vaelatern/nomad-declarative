package templating

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// getArg retrieves a value from the `data` map using `in` as the key.
// If the value is a string, it is returned directly; otherwise, it is converted into HCL format.
func getArg(in string, data map[string]interface{}) (string, error) {
	key := in

	val, exists := data[key]
	if !exists {
		return "", fmt.Errorf("key %q not found in data", key)
	}

	// Convert the value to an HCL representation
	hclStr, err := convertToHCL(val)
	if err != nil {
		return "", err
	}

	return (hclStr), nil
}

// convertToHCL converts a Go value into its natural HCL representation
func convertToHCL(v interface{}) (string, error) {
	// Convert the Go value to a cty.Value
	ctyVal, err := gocty.ToCtyValue(v, inferCtyType(v))
	if err != nil {
		return "", err
	}

	// Create a new HCL file to write the value
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	// Write the value in HCL format
	rootBody.SetAttributeValue("value", ctyVal)

	// Extract the HCL representation from the file
	hclBytes := f.Bytes()
	hclOutput := string(hclBytes)

	// Remove the "value =" prefix to return only the raw HCL representation
	return strings.Trim(hclOutput[7:], "\n "), nil
}

// inferCtyType determines the cty.Type of a given Go value
func inferCtyType(v interface{}) cty.Type {
	switch val := v.(type) {
	case string:
		return cty.String
	case int, int32, int64, float32, float64:
		return cty.Number
	case bool:
		return cty.Bool
	case []interface{}:
		if len(val) > 0 {
			return cty.ListValEmpty(inferCtyType(val[0])).Type()
		}
		return cty.List(cty.DynamicPseudoType)
	case map[string]interface{}:
		objType := make(map[string]cty.Type)
		for k, v := range val {
			objType[k] = inferCtyType(v)
		}
		return cty.Object(objType)
	default:
		return cty.DynamicPseudoType
	}
}
