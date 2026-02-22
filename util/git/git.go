package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/IronFE/iron.cli/util"
)

func GetRootDir(workDir string) (string, error) {
	return util.Run(workDir, "git", "rev-parse", "--show-toplevel")
}

func OriginRemote(workDir string) (string, error) {
	output, err := util.Run(workDir, "git", "remote", "-v")
	if err != nil {
		return "", fmt.Errorf("failed to find git remote on %s: %w", workDir, err)
	}

	exp := regexp.MustCompile("^origin\\t(?P<repo>[\\w@.:\\/]+\\.git) \\((push|fetch)\\)$")

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := exp.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		return matches[exp.SubexpIndex("repo")], nil
	}

	return "", fmt.Errorf("no origin branch found in repo %s", workDir)
}
