// Copyright Contributors to the Open Cluster Management project

package dryrun

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"open-cluster-management.io/config-policy-controller/pkg/mappings"
)

type DryRunner struct {
	policyPath   string
	messagesPath string
	printDiffs   bool
	statusPath   string
	mappingsPath string
	logPath      string
}

var ErrNonCompliant = errors.New("policy is NonCompliant")

func (d *DryRunner) GetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dryrun",
		Short: "(Dev Preview feature) Locally execute a ConfigurationPolicy",
		Long: "(Dev Preview feature) Locally execute a ConfigurationPolicy against input files " +
			"representing the cluster state, and view the diffs and any compliance events that " +
			"would be generated.",
		RunE: d.dryRun,
		Args: cobra.ArbitraryArgs,
	}

	cmd.Flags().StringVarP(
		&d.policyPath,
		"policy",
		"p",
		"",
		"The input Policy or ConfigurationPolicy to execute",
	)

	if err := cmd.MarkFlagRequired("policy"); err != nil {
		panic(err)
	}

	cmd.Flags().StringVar(
		&d.messagesPath,
		"messages-path",
		"",
		"An optional file to save the compliance messages emitted by the policy, "+
			"with one message per line. If not set, messages will be printed.",
	)

	cmd.Flags().BoolVar(
		&d.printDiffs,
		"print-diffs",
		true,
		"Set to false to omit any diffs generated by the policy.",
	)

	cmd.Flags().StringVar(
		&d.statusPath,
		"status-path",
		"",
		"An optional file to save the full resulting status of the policy.",
	)

	mappingsPath := os.Getenv("DRYRUN_MAPPINGS_FILE") // empty if not set

	cmd.Flags().StringVar(
		&d.mappingsPath,
		"mappings-file",
		mappingsPath,
		"An optional set of API Mappings to use. If omitted a default set will "+
			"be used. A compatible file may be created by the 'generate' subcommand, "+
			"and might be necessary if using custom resources. Can also be set via "+
			"the DRYRUN_MAPPINGS_FILE environment variable.",
	)

	cmd.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Generate an API Mappings file",
		Long: "Connects to the default kubernetes cluster (configurable via " +
			"the usual patterns like KUBECONFIG), discovers its api-resources, " +
			"and saves them to a JSON file in a structure readable by the " +
			"config-policy-controller.",
		RunE: mappings.GenerateMappings,
	})

	cmd.SetOut(os.Stdout) // sets default output to stdout, otherwise it is stderr

	return cmd
}

func Execute() error {
	cmd := DryRunner{}

	err := cmd.GetCmd().Execute()

	if errors.Is(err, ErrNonCompliant) {
		os.Exit(2) // Special exit code to distinguish non-compliance from other errors
	}

	return err
}
