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

var petsCmd = &cobra.Command{
	Use:   "pets",
	Short: "Pets operations",
}

func init() {
	rootCmd.AddCommand(petsCmd)

	// createPet
	{

		cmd := &cobra.Command{
			Use:   "create-pet",
			Short: "Create a pet",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
		}

		petsCmd.AddCommand(cmd)
	}

	// deletePet
	{
		var flagpetId string

		cmd := &cobra.Command{
			Use:   "delete-pet",
			Short: "Delete a pet",
			RunE: func(cmd *cobra.Command, args []string) error {
				query := url.Values{}
				path := "/pets/{petId}"

				path = strings.Replace(path, "{pet-id}", fmt.Sprintf("%v", flagpetId), 1)

				_ = query
				_ = path
				return nil
			},
		}

		cmd.Flags().StringVar(&flagpetId, "pet-id", "", "")

		_ = cmd.MarkFlagRequired("pet-id")

		petsCmd.AddCommand(cmd)
	}

	// getPet
	{
		var flagpetId string

		cmd := &cobra.Command{
			Use:   "get-pet",
			Short: "Get a pet by ID",
			RunE: func(cmd *cobra.Command, args []string) error {
				query := url.Values{}
				path := "/pets/{petId}"

				path = strings.Replace(path, "{pet-id}", fmt.Sprintf("%v", flagpetId), 1)

				_ = query
				_ = path
				return nil
			},
		}

		cmd.Flags().StringVar(&flagpetId, "pet-id", "", "")

		_ = cmd.MarkFlagRequired("pet-id")

		petsCmd.AddCommand(cmd)
	}

	// listPets
	{
		var flaglimit int32
		var flagstatus string

		cmd := &cobra.Command{
			Use:   "list-pets",
			Short: "List all pets",
			RunE: func(cmd *cobra.Command, args []string) error {
				query := url.Values{}
				path := "/pets"

				if cmd.Flags().Changed("limit") {
					query.Set("limit", fmt.Sprintf("%v", flaglimit))
				}

				if cmd.Flags().Changed("status") {
					query.Set("status", fmt.Sprintf("%v", flagstatus))
				}

				_ = query
				_ = path
				return nil
			},
		}

		cmd.Flags().Int32Var(&flaglimit, "limit", 0, "")

		cmd.Flags().StringVar(&flagstatus, "status", "", "")

		petsCmd.AddCommand(cmd)
	}

}
