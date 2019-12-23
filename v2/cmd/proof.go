package cmd

import (
	"github.com/Defacto2/uuid/v2/lib/database"
	"github.com/Defacto2/uuid/v2/lib/logs"
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
		err := database.CreateProof()
		logs.Check(err)
		// f := ex[2]
		// name := "/Users/ben/Downloads/" + f
		// fmt.Println("File: ", f)
		// l := strings.Join(archive.ReadArchive(name), ",")
		// fmt.Println("Contains content: ", l)
		// archive.ExtractArchive(name)
	},
}

func init() {
	rootCmd.AddCommand(proofCmd)
}
