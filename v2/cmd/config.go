package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/Defacto2/uuid/v2/lib/logs"
	"github.com/alecthomas/chroma/quick"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	configSetName string
	fileOverwrite bool
	infoStyles    string
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure defaults",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			_ = cmd.Usage()
			os.Exit(0)
		}
		_ = cmd.Usage()
		logs.Check(fmt.Errorf("invalid command %v please use one of the available config commands", args[0]))
	},
}

var configCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new config file",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg := viper.ConfigFileUsed(); cfg != "" && !fileOverwrite {
			configExists(cmd.CommandPath(), "create")
		}
		writeConfig(false)
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove the config file",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := viper.ConfigFileUsed()
		if cfg == "" {
			configMissing(cmd.CommandPath(), "delete")
		}
		if _, err := os.Stat(cfg); os.IsNotExist(err) {
			configMissing(cmd.CommandPath(), "delete")
		}
		switch promptYN("Confirm the file deletion", false) {
		case true:
			logs.Check(fmt.Errorf("Could not remove %v %v", cfg, os.Remove(cfg)))
			fmt.Println("The file is deleted")
		}
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the config file",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := viper.ConfigFileUsed()
		if cfg == "" {
			configMissing(cmd.CommandPath(), "edit")
		}
		var edit string
		if err := viper.BindEnv("editor", "EDITOR"); err != nil {
			editors := []string{"nano", "vim", "emacs"}
			if runtime.GOOS == "windows" {
				editors = append(editors, "notepad++.exe", "notepad.exe")
			}
			for _, editor := range editors {
				if _, err := exec.LookPath(editor); err == nil {
					edit = editor
					break
				}
			}
			if edit != "" {
				fmt.Printf("There is no %s environment variable set so using: %s\n", "EDITOR", edit)
			}
		} else {
			edit = viper.GetString("editor")
			if _, err := exec.LookPath(edit); err != nil {
				logs.Check(fmt.Errorf("%v command not found %v", edit, exec.ErrNotFound))
			}
		}
		// credit: https://stackoverflow.com/questions/21513321/how-to-start-vim-from-go
		exe := exec.Command(edit, cfg)
		exe.Stdin = os.Stdin
		exe.Stdout = os.Stdout
		if err := exe.Run(); err != nil {
			fmt.Printf("%s\n", err)
		}
	},
}

var configInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "View settings configured by the config",
	Run: func(cmd *cobra.Command, args []string) {
		println("These are the default configurations used by the commands of RetroTxt when no flags are given.\n")
		sets, err := yaml.Marshal(viper.AllSettings())
		logs.Check(err)
		if err := quick.Highlight(os.Stdout, string(sets), "yaml", "terminal256", infoStyles); err != nil {
			fmt.Println(string(sets))
		}
		println()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change a configuration",
	//todo add long with information on how to view settings
	Example: `--name create.meta.description # to change the meta description setting
--name version.format          # to set the version command output format`,
	Run: func(cmd *cobra.Command, args []string) {
		var name = configSetName
		keys := viper.AllKeys()
		sort.Strings(keys)
		// sort.SearchStrings() - The slice must be sorted in ascending order.
		if i := sort.SearchStrings(keys, name); i == len(keys) || keys[i] != name {
			err := fmt.Errorf("to see a list of usable settings, run: retrotxt config info")
			logs.Check(fmt.Errorf("invalid flag %v %v", fmt.Sprintf("--name %s", name), err))
		}
		s := viper.GetString(name)
		switch s {
		case "":
			fmt.Printf("\n%s is currently disabled\n", name)
		default:
			fmt.Printf("\n%s is currently set to %q\n", name, s)
		}
		// switch {
		// case name == "create.layout":
		// 	fmt.Printf("Set a new value, choices: %s\n", ci(createLayouts()))
		// 	promptSlice(createLayouts())
		// case name == "info.format":
		// 	fmt.Printf("Set a new value, choice: %s\n", ci(infoFormats))
		// 	promptSlice(infoFormats)
		// case name == "version.format":
		// 	fmt.Printf("Set a new value, choices: %s\n", ci(versionFormats))
		// 	promptSlice(versionFormats)
		// case name == "create.server-port":
		// 	fmt.Printf("Set a new HTTP port, choices: %v-%v (recommended: 8080)\n", portMin, portMax)
		// 	promptPort()
		// case name == "create.meta.generator":
		// 	fmt.Printf("<meta name=\"generator\" content=\"RetroTxt v%s\">\nEnable this element? [y/n]\n", GoBuildVer)
		// 	promptBool()
		// case s == "":
		// 	promptMeta(s)
		// 	fmt.Printf("\nSet a new value or leave blank to keep it disabled: \n")
		// 	promptString(s)
		// default:
		// 	promptMeta(s)
		// 	fmt.Printf("\nSet a new value, leave blank to keep as-is or use a dash [-] to disable: \n")
		// 	promptString(s)
		// }
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configCreateCmd)
	configCreateCmd.Flags().BoolVarP(&fileOverwrite, "overwrite", "y", false, "overwrite any existing config file")
	configCmd.AddCommand(configDeleteCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInfoCmd)
	configInfoCmd.Flags().StringVarP(&infoStyles, "syntax-style", "c", "monokai", "config syntax highligher, \"use none\" to disable")
	configCmd.AddCommand(configSetCmd)
	configSetCmd.Flags().StringVarP(&configSetName, "name", "n", "", `the configuration path to edit in dot syntax (see examples)
to see a list of names run: retrotxt config info`)
	_ = configSetCmd.MarkFlagRequired("name")
}

func configExists(name string, suffix string) {
	cmd := strings.TrimSuffix(name, suffix)
	fmt.Printf("A config file already is in use at: %s\n", viper.ConfigFileUsed())
	fmt.Printf("To edit it: %s\n", cmd+"edit")
	fmt.Printf("To delete:  %s\n", cmd+"delete")
	os.Exit(1)
}

func configMissing(name string, suffix string) {
	cmd := strings.TrimSuffix(name, suffix) + "create"
	fmt.Printf("No config file is in use.\nTo create one run: %s\n", cmd)
	os.Exit(1)
}

func promptYN(query string, yesDefault bool) bool {
	var input string
	y := "Y"
	n := "n"
	if !yesDefault {
		y = "y"
		n = "N"
	}
	fmt.Printf("%s? [%s/%s] ", query, y, n)
	fmt.Scanln(&input)
	switch input {
	case "":
		if yesDefault {
			return true
		}
	case "yes", "y":
		return true
	}
	return false
}

func writeConfig(update bool) {
	bs, err := yaml.Marshal(viper.AllSettings())
	logs.Check(err)
	d, err := os.UserHomeDir()
	logs.Check(err)
	err = ioutil.WriteFile(d+"/.df2.yaml", bs, 0660)
	logs.Check(err)
	s := "Created a new"
	if update {
		s = "Updated the"
	}
	fmt.Println(s+" config file at:", d+"/.df2.yaml")
}
