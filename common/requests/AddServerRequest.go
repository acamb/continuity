package requests

import (
	"continuity/common"
	"continuity/server/loadbalancer"
	"net/url"
)

type AddServerRequest struct {
	NewServerAddress string           `json:"new_server_address" binding:"required"`
	Condition        common.Condition `json:"condition"`
	HealthCheckPath  string           `json:"health_check_path" binding:"required"`
}

func (req *AddServerRequest) Validate() (*loadbalancer.ServerHost, error) {
	if req.Condition != (common.Condition{}) {
		err := req.Condition.Validate()
		if err != nil {
			return nil, err
		}
	}
	parsed, err := url.Parse(req.NewServerAddress)
	if err != nil {
		return nil, err
	}
	return loadbalancer.NewServerHost(parsed.String(), req.HealthCheckPath, req.Condition)
}
