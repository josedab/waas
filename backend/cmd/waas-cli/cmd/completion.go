package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for WAAS CLI.

To load completions:

Bash:
  $ source <(waas completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ waas completion bash > /etc/bash_completion.d/waas
  # macOS:
  $ waas completion bash > $(brew --prefix)/etc/bash_completion.d/waas

Zsh:
  $ source <(waas completion zsh)
  # To load completions for each session, execute once:
  $ waas completion zsh > "${fpath[1]}/_waas"

Fish:
  $ waas completion fish | source
  # To load completions for each session, execute once:
  $ waas completion fish > ~/.config/fish/completions/waas.fish

PowerShell:
  PS> waas completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, execute once:
  PS> waas completion powershell > waas.ps1
  # and source this file from your PowerShell profile.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
