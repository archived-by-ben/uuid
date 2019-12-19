package cmd

import (
	"sort"
	"strings"

	"github.com/Defacto2/uuid/v2/lib/assets"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Discover or clean orphan files",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		assets.Init(makeDirs)
		assets.Clean(delete, humanize, result, target)
	},
}

var (
	delete   bool
	humanize bool
	makeDirs bool
	result   string
	results  []string = []string{"text", "quiet"}
	target   string
	targets  []string = []string{"all", "download", "emulation", "image"}
)

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().StringVarP(&target, "target", "t", "all", "what file section to clean"+options(targets))
	cleanCmd.Flags().StringVarP(&result, "result", "r", "text", "print format for the results of clean"+options(results))
	cleanCmd.Flags().BoolVarP(&delete, "delete", "d", false, "erase all discovered files to free up drive space")
	cleanCmd.Flags().BoolVar(&humanize, "humanize", true, "humanize file sizes and date times")
	cleanCmd.Flags().BoolVar(&makeDirs, "makedirs", false, "generate uuid directories and placeholder files")
	cleanCmd.Flags().SortFlags = false
	_ = cleanCmd.Flags().MarkHidden("makedirs")
}

func options(a []string) string {
	sort.Strings(a)
	return "\noptions: " + strings.Join(a, ", ")
}

func valid(a []string, x string) bool {
	for _, b := range a {
		if b == x {
			return true
		}
	}
	return false
}
