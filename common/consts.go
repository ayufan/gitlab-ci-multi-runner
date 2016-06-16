package common

import "time"

const DefaultTimeout = 7200
const DefaultExecTimeout = 1800
const CheckInterval = 3 * time.Second
const NotHealthyCheckInterval = 300
const UpdateInterval = 3 * time.Second
const UpdateRetryInterval = 3 * time.Second
const ReloadConfigInterval = 3
const HealthyChecks = 3
const HealthCheckInterval = 3600
const DefaultWaitForServicesTimeout = 30
const ShutdownTimeout = 30
const DefaultOutputLimit = 4096 // 4MB in kilobytes
const ForceTraceSentInterval = 30 * time.Second
