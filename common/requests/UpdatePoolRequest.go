package requests

type UpdatePoolRequest struct {
	Hostname                string `json:"hostname" binding:"required"`
	HealthCheckInterval     int64  `json:"health_check_interval" validate:"gt=0"`
	HealthCheckInitialDelay int64  `json:"health_check_initial_delay" validate:"gt=0"`
	HealthCheckTimeout      int64  `json:"health_check_timeout" validate:"gt=0"`
	HealthCheck_numOk       uint32 `json:"health_check_num_ok"`
	HealthCheck_numFail     uint32 `json:"health_check_num_fail"`
}
