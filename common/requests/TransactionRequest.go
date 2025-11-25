package requests

import (
	"continuity/common"
	"continuity/server/loadbalancer"
	"errors"
	"net/url"
)

type TransactionRequest struct {
	NewServerAddress         string           `json:"new_server_address" binding:"required"`
	NewServerCondition       common.Condition `json:"new_server_condition"`
	NewServerHealthCheckPath string           `json:"new_server_health_check_path" binding:"required"`
	OldServerId              string           `json:"old_server_id" binding:"required"`
}

func (req *TransactionRequest) Validate() (*loadbalancer.ServerHost, error) {

	if req.NewServerCondition != (common.Condition{}) {
		err := req.NewServerCondition.Validate()
		if err != nil {
			return nil, err
		}
	}
	if req.NewServerHealthCheckPath == "" {
		return nil, errors.New("health_check_path is required")
	}
	parsed, err := url.Parse(req.NewServerAddress)
	if err != nil {
		return nil, err
	}
	return loadbalancer.NewServerHost(parsed.String(), req.NewServerHealthCheckPath, req.NewServerCondition)
}
