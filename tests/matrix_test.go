package tests

import (
	archive "github.com/osteele/matrix-archive/lib"
	"testing"
)

func TestMatrixFunctionsAccessible(t *testing.T) {
	// Test that Matrix-related functions are accessible from the archive package
	// We don't actually call them since they require proper setup

	// Test BeeperAuth exists and can be created
	auth := archive.NewBeeperAuth("")
	if auth == nil {
		t.Error("NewBeeperAuth should return a valid instance")
	}

	// Test that we can access the exported methods
	_ = auth.LoadCredentials
	_ = auth.SaveCredentials
	_ = auth.GetCredentialsFilePath
	_ = auth.SaveCredentialsToFile
	_ = auth.LoadCredentialsFromFile
}
