package provider

import "github.com/spf13/cobra"

// FlagInteractive is the persistent flag that opts into interactive prompts
// (e.g. building a request body via a form instead of passing raw JSON).
const FlagInteractive = "interactive"

// RegisterInteractiveFlag registers the persistent --interactive/-i flag.
func RegisterInteractiveFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolP(FlagInteractive, "i", false, "interactively prompt for input")
}

// Interactive reports whether the --interactive/-i flag is set on the command
// or any of its parents.
func Interactive(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup(FlagInteractive)
	if flag == nil {
		return false
	}
	value, err := cmd.Flags().GetBool(FlagInteractive)
	return err == nil && value
}
