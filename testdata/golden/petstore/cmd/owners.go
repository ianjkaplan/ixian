package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Ensure imports are used.
var (
	_ = json.Marshal
	_ = fmt.Sprintf
	_ = url.Values{}
	_ = os.Stdout
	_ = strings.Replace
)

var ownersCmd = &cobra.Command{
	Use:   "owners",
	Short: "Owners operations",
}

func init() {
	rootCmd.AddCommand(ownersCmd)

	// listOwners
	{

		cmd := &cobra.Command{
			Use:   "list-owners",
			Short: "List all owners",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
		}

		ownersCmd.AddCommand(cmd)
	}

}
