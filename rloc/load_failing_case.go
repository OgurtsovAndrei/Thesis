package rloc

import (
	"Thesis/bits"
	"encoding/json"
	"fmt"
	"os"
)

// LoadFailingCase loads a saved failing test case from a JSON file
func LoadFailingCase(filename string) ([]bits.BitString, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var failCase FailingCase
	err = json.Unmarshal(data, &failCase)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	keys := make([]bits.BitString, failCase.NumKeys)
	for i := 0; i < failCase.NumKeys; i++ {
		keys[i] = bits.NewCharBitStringFromDataAndSize(failCase.KeysData[i], failCase.KeysSizes[i])
	}

	fmt.Printf("Loaded failing case with %d keys (seed: %d)\n", failCase.NumKeys, failCase.Seed)
	fmt.Printf("Original error: %s\n", failCase.Error)

	return keys, nil
}
