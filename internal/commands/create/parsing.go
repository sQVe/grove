package create

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/sqve/grove/internal/errors"
)

func ParseCreateOptions(cmd *cobra.Command, args []string) (CreateOptions, error) {
	options := CreateOptions{
		BranchName: args[0],
		// Set a dummy progress callback for testing.
		ProgressCallback: func(message string) {
			// Do nothing in test context.
		},
	}

	if len(args) > 1 {
		options.WorktreePath = args[1]
	}

	if err := parseBasicFlags(cmd, &options); err != nil {
		return options, err
	}

	if err := parseCopyFlags(cmd, &options); err != nil {
		return options, err
	}

	return options, nil
}

func parseBasicFlags(cmd *cobra.Command, options *CreateOptions) error {
	var err error

	if options.BaseBranch, err = cmd.Flags().GetString("base"); err != nil {
		return errors.Wrap(err, "failed to get base flag")
	}

	return nil
}

func parseCopyFlags(cmd *cobra.Command, options *CreateOptions) error {
	var err error

	if options.CopyEnv, err = cmd.Flags().GetBool("copy-env"); err != nil {
		return errors.Wrap(err, "failed to get copy-env flag")
	}

	if copyPatterns, err := cmd.Flags().GetString("copy"); err != nil {
		return errors.Wrap(err, "failed to get copy flag")
	} else if copyPatterns != "" {
		options.CopyPatterns = parseCopyPatterns(copyPatterns)
	}

	noCopy, err := cmd.Flags().GetBool("no-copy")
	if err != nil {
		return errors.Wrap(err, "failed to get no-copy flag")
	}

	// Set default patterns for copy-env if no explicit patterns were provided.
	if options.CopyEnv && len(options.CopyPatterns) == 0 {
		options.CopyPatterns = []string{".env*", "*.local.*", "docker-compose.override.yml"}
	}

	options.CopyFiles = determineCopyBehavior(noCopy, options.CopyEnv, len(options.CopyPatterns) > 0)
	return nil
}

func parseCopyPatterns(patterns string) []string {
	parts := strings.Split(patterns, ",")
	for i, pattern := range parts {
		parts[i] = strings.TrimSpace(pattern)
	}
	return parts
}

func determineCopyBehavior(noCopy, copyEnv, hasCopyPatterns bool) bool {
	if noCopy {
		return false
	}
	return copyEnv || hasCopyPatterns
}
