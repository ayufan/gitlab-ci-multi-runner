package common

const repoURL = "https://gitlab.com/gitlab-org/gitlab-test.git"
const repoSHA = "6907208d755b60ebeacb2e9dfea74c92c3449a1f"
const repoBeforeSHA = "c347ca2e140aa667b968e51ed0ffe055501fe4f4"
const repoRefName = "master"

var SuccessfulBuild = GetBuildResponse{
	RepoURL:   repoURL,
	Commands:  "echo Hello World",
	Sha:       repoSHA,
	BeforeSha: repoBeforeSHA,
	RefName:   repoRefName,
}

var FailedBuild = GetBuildResponse{
	RepoURL:   repoURL,
	Commands:  "exit 1",
	Sha:       repoSHA,
	BeforeSha: repoBeforeSHA,
	RefName:   repoRefName,
}

var LongRunningBuild = GetBuildResponse{
	RepoURL:   repoURL,
	Commands:  "sleep 3600",
	Sha:       repoSHA,
	BeforeSha: repoBeforeSHA,
	RefName:   repoRefName,
}
