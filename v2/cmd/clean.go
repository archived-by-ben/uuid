/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"sort"
	"strings"

	data "github.com/Defacto2/uuid/v2/lib"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Discover or clean orphan UUID named files",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("clean called")
		// println(valid(targets, target))
		// _, m := data.CreateUUIDMap()

		// for k, v := range m {
		// 	fmt.Printf("key[%s] value[%s]\n", k, v)
		// }

		// println("count", len(m))

		data.Init()
		data.Clean()
	},
}

var (
	delete   bool
	humanize bool
	result   string
	results  []string = []string{"text", "json", "quiet"}
	target   string
	targets  []string = []string{"all", "download", "emulation", "image"}
)

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

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&delete, "delete", "d", false, "erase all discovered files to free up drive space")
	cleanCmd.Flags().BoolVar(&humanize, "humanize", true, "humanize file sizes and date times")
	cleanCmd.Flags().StringVarP(&result, "result", "r", "text", "print format for the results of clean"+options(results))
	cleanCmd.Flags().StringVarP(&target, "target", "t", "all", "what file section to clean"+options(targets))
}
