package common

import "time"

const DefaultTimeout = 7200
const CheckInterval = 3
const NotHealthyCheckInterval = 300
const UpdateInterval = 3
const UpdateRetryInterval = 3
const ReloadConfigInterval = 3
const HealthyChecks = 3
const HealthCheckInterval = 3600
const DefaultWaitForServicesTimeout = 30
const ShutdownTimeout = 30
const MaxTraceOutputSize = 1024 * 1024 // 1MB
const ForceTraceSentInterval time.Duration = 300
