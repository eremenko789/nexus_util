package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssetCmd(t *testing.T) {
	assert.NotNil(t, AssetCmd)
	assert.Equal(t, "asset", AssetCmd.Use)
	assert.Equal(t, "Asset management commands", AssetCmd.Short)
	assert.NotEmpty(t, AssetCmd.Long)

	// Verify that subcommands are registered
	assert.NotNil(t, ListCmd)
	assert.NotNil(t, PushCmd)
	assert.NotNil(t, PullCmd)
	assert.NotNil(t, DeleteCmd)
}
