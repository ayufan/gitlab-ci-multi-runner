package shells

import (
	"fmt"
	. "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type AbstractShell struct {
}

func (s *AbstractShell) GetDefaultVariables(build *Build, projectDir string) []string {
	return []string{
		fmt.Sprintf("CI_BUILD_REF=%s", build.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", build.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", build.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", build.ID),
		fmt.Sprintf("CI_BUILD_REPO=%s", build.RepoURL),
		fmt.Sprintf("CI_PROJECT_ID=%d", build.ProjectID),
		fmt.Sprintf("CI_PROJECT_DIR=%s", projectDir),
		"CI=true",
		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",
		"GITLAB_CI=true",
	}
}

func (s *AbstractShell) GetBuildVariables(build *Build) []string {
	var variables []string
	for _, variable := range build.Variables {
		variables = append(variables,
			fmt.Sprintf("%s=%s", variable.Key, variable.Value))
	}
	return variables
}

func (s *AbstractShell) GetVariables(build *Build, projectDir string) []string {
	variables := s.GetDefaultVariables(build, projectDir)
	variables = append(variables, s.GetBuildVariables(build)...)
	return variables
}
