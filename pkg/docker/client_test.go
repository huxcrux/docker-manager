package docker

import (
	"testing"
)

func TestCreateClient(t *testing.T) {
	_, err := CreateClient()
	if err != nil {
		t.Errorf("Failed to create Docker client: %v", err)
	}

	// Add additional test cases here if needed
}
