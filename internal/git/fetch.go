package git

import (
	"bufio"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sqve/grove/internal/logger"
)

type ChangeType int

const (
	New ChangeType = iota
	Updated
	Pruned
)

func (c ChangeType) String() string {
	switch c {
	case New:
		return "new"
	case Updated:
		return "updated"
	case Pruned:
		return "pruned"
	default:
		return "unknown"
	}
}

type RefChange struct {
	RefName string
	OldHash string
	NewHash string
	Type    ChangeType
}

func DetectRefChanges(before, after map[string]string) []RefChange {
	var changes []RefChange

	for refName, newHash := range after {
		oldHash, exists := before[refName]
		if !exists {
			changes = append(changes, RefChange{
				RefName: refName,
				OldHash: "",
				NewHash: newHash,
				Type:    New,
			})
		} else if oldHash != newHash {
			changes = append(changes, RefChange{
				RefName: refName,
				OldHash: oldHash,
				NewHash: newHash,
				Type:    Updated,
			})
		}
	}

	for refName, oldHash := range before {
		if _, exists := after[refName]; !exists {
			changes = append(changes, RefChange{
				RefName: refName,
				OldHash: oldHash,
				NewHash: "",
				Type:    Pruned,
			})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].RefName < changes[j].RefName
	})

	return changes
}

func GetRemoteRefs(repoPath, remote string) (map[string]string, error) {
	if repoPath == "" {
		return nil, errors.New("repository path cannot be empty")
	}
	if remote == "" {
		return nil, errors.New("remote name cannot be empty")
	}

	refPattern := fmt.Sprintf("refs/remotes/%s/", remote)
	logger.Debug("Executing: git for-each-ref --format=%%(refname) %%(objectname) %s in %s", refPattern, repoPath)

	cmd, cancel := GitCommand("git", "for-each-ref", "--format=%(refname) %(objectname)", refPattern) //nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	output, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote refs: %w", err)
	}

	refs := make(map[string]string)
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 2 {
			refs[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse remote refs: %w", err)
	}

	return refs, nil
}

func FetchRemote(repoPath, remote string) error {
	if repoPath == "" {
		return errors.New("repository path cannot be empty")
	}
	if remote == "" {
		return errors.New("remote name cannot be empty")
	}

	logger.Debug("Executing: git fetch --prune %s in %s", remote, repoPath)
	cmd, cancel := GitCommand("git", "fetch", "--prune", remote) //nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

func CountCommits(repoPath, fromHash, toHash string) int {
	cmd, cancel := GitCommand("git", "rev-list", "--count", fromHash+".."+toHash) //nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	output, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return 0
	}

	line := strings.TrimSpace(output.String())
	var count int
	if _, err := fmt.Sscanf(line, "%d", &count); err != nil {
		return 0
	}

	return count
}
