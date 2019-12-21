package cmd

import (
	"fmt"
	"strings"

	"github.com/Defacto2/uuid/v2/lib/database"
	"github.com/spf13/cobra"
)

var ex = map[uint]string{
	0: "Yo-Kai_Watch_2_Psychic_Specters_EUR_MULTi7-TRSI.zip",
	1: "Miitopia_EUR_MULTi6-TRSI.zip",
	2: "Hey_Pikmin_EUR_MULTi6-TRSI.zip",
}

// proofCmd represents the proof command
var proofCmd = &cobra.Command{
	Use:   "proof",
	Short: "Batch handler for #proof tagged files",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		f := ex[0]
		name := "/Users/ben/Downloads/" + f
		fmt.Println("File: ", f)
		//		database.CreateProof()
		l := strings.Join(database.ReadArchive(name), ",")
		fmt.Println("Contains content: ", l)

		database.ExtractArchive(name)
	},
}

func init() {
	rootCmd.AddCommand(proofCmd)
}
