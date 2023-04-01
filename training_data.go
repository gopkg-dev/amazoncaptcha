package amazoncaptcha

import (
	_ "embed"
	"encoding/json"
)

// Embed the training data file as a byte slice using the embed package

//go:embed training_data.json
var data []byte

// featureMap is a map that stores training data with string keys and values.
// WARNING: featureMap is not safe for concurrent modification.
// It should only be accessed for reading in a concurrent setting.
var featureMap map[string]string

// Define an init function to run at module initialization time
func init() {
	// Unmarshal the training data from the embedded byte slice into the map
	_ = json.Unmarshal(data, &featureMap)
}
