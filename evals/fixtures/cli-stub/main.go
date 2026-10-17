// cairn: feature flags for small teams, from the terminal. A deliberately plain cobra
// CLI with no TUI yet: the "before" for the init-interview eval. Fictional product.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var envs = []string{"dev", "staging", "prod"}

func main() {
	root := &cobra.Command{
		Use:   "cairn",
		Short: "Feature flags for small teams, from the terminal",
		Long: `cairn keeps one flag file per environment in a Git repo and lets you list,
toggle and roll out flags without opening a web console.`,
	}

	flags := &cobra.Command{Use: "flags", Short: "List, toggle and roll out flags"}
	var env string
	list := &cobra.Command{
		Use:   "list",
		Short: "Every flag with its state per environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%-24s %-8s %-8s %-8s\n", "FLAG", "DEV", "STAGING", "PROD")
			fmt.Printf("%-24s %-8s %-8s %-8s\n", "new-checkout", "on", "on", "12%")
			fmt.Printf("%-24s %-8s %-8s %-8s\n", "dark-mode", "on", "off", "off")
			return nil
		},
	}
	list.Flags().StringVar(&env, "env", "", "only this environment")
	toggle := &cobra.Command{
		Use:   "toggle <name>",
		Short: "Flip a flag in one environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if env == "" {
				return fmt.Errorf("--env is required (one of %v)", envs)
			}
			fmt.Printf("%s: toggled in %s\n", args[0], env)
			return nil
		},
	}
	toggle.Flags().StringVar(&env, "env", "", "the environment to change")
	var percent int
	rollout := &cobra.Command{
		Use:   "rollout <name>",
		Short: "Ramp a flag to a share of users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%s: %d%% in %s\n", args[0], percent, env)
			return nil
		},
	}
	rollout.Flags().StringVar(&env, "env", "prod", "the environment to change")
	rollout.Flags().IntVar(&percent, "percent", 0, "share of users, 0-100")
	flags.AddCommand(list, toggle, rollout)

	envsCmd := &cobra.Command{
		Use:   "envs",
		Short: "The environments and their last change",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, e := range envs {
				fmt.Printf("%-10s last change 2h ago\n", e)
			}
			return nil
		},
	}

	root.AddCommand(flags, envsCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
