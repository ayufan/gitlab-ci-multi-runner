package docker

import "time"

const dockerAPIVersion = "1.18"
const dockerImageTTL = 6 * time.Minute
const dockerLabelPrefix = "com.gitlab.gitlab-runner"