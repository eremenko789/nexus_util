package asset

import (
	"github.com/spf13/cobra"
)

var AssetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Asset management commands",
	Long:  "Commands for managing assets (files and directories) in Nexus repository",
}

