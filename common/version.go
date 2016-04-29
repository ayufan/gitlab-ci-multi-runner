package common

import (
	"time"
)

var NAME = "gitlab-ci-multi-runner"
var VERSION = "dev"
var REVISION = "HEAD"
var BUILT = time.Now().Format(time.RFC1123Z)
