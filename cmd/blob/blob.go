package blob

import (
	"github.com/spf13/cobra"
)

var BlobCmd = &cobra.Command{
	Use:   "blob",
	Short: "Blob store management commands",
	Long:  "Commands for managing Nexus blob stores",
}
