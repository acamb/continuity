package requests

import (
	"continuity/server/loadbalancer"
	"errors"
	_ "github.com/go-playground/validator/v10"
	"maps"
	"time"
)

type CreatePoolRequest struct {
	Hostname                string `json:"hostname" binding:"required"`
	HealthCheckInterval     int64  `json:"health_check_interval" binding:"required" validate:"gt=0"`
	HealthCheckInitialDelay int64  `json:"health_check_initial_delay" binding:"required" validate:"gt=0"`
	HealthCheckTimeout      int64  `json:"health_check_timeout" binding:"required" validate:"gt=0"`
	HealthCheck_numOk       uint32 `json:"health_check_num_ok" binding:"required" validate:"gt=0"`
	HealthCheck_numFail     uint32 `json:"health_check_num_fail" binding:"required" validate:"gt=0"`
	StickySessions          bool   `json:"sticky_sessions"`
	StickyMethod            string `json:"sticky_method"`
	StickySessionTimeout    int64  `json:"sticky_session_timeout"`
	StickySessionCookieName string `json:"sticky_session_cookie_name"`
}

func (req *CreatePoolRequest) Validate() (*loadbalancer.Pool, error) {
	var pool *loadbalancer.Pool
	if req.StickySessions {
		if req.StickyMethod == "" {
			return nil, errors.New("sticky_method is required when sticky_sessions is true")
		}
		stickyMethod, err := loadbalancer.GetStickyMethodFromString(req.StickyMethod)
		if err != nil {
			methods := ""
			for m := range maps.Values(loadbalancer.StickyMethodName) {
				if methods != "" {
					methods += ", "
				}
				methods += m
			}
			return nil, errors.New("invalid sticky_method, possible values are: " + methods)
		}
		if req.StickySessionTimeout <= 0 {
			return nil, errors.New("sticky_session_timeout must be greater than 0 when sticky_sessions is true")
		}
		switch stickyMethod {
		case loadbalancer.StickyMethod_AppCookie:
			if req.StickySessionCookieName == "" {
				return nil, errors.New("sticky_session_cookie_name is required")
			}
			if req.StickySessionTimeout <= 0 {
				return nil, errors.New("sticky_session_cookie_name must be greater than 0")
			}
			pool, err = loadbalancer.NewPoolWithStickySessionCustomCookie(
				req.Hostname,
				time.Duration(req.HealthCheckTimeout*int64(time.Second)),
				time.Duration(req.HealthCheckInterval*int64(time.Second)),
				time.Duration(req.HealthCheckInitialDelay*int64(time.Second)),
				time.Duration(req.StickySessionTimeout*int64(time.Second)),
				req.HealthCheck_numOk,
				req.HealthCheck_numFail,
				req.StickySessionCookieName,
			)
			break
		case loadbalancer.StickyMethod_IP:
			if req.StickySessionTimeout <= 0 {
				return nil, errors.New("sticky_session_cookie_name must be greater than 0")
			}
			pool = loadbalancer.NewPoolWithIPStickySessions(
				req.Hostname,
				time.Duration(req.HealthCheckTimeout*int64(time.Second)),
				time.Duration(req.HealthCheckInterval*int64(time.Second)),
				time.Duration(req.HealthCheckInitialDelay*int64(time.Second)),
				time.Duration(req.StickySessionTimeout*int64(time.Second)),
				req.HealthCheck_numOk,
				req.HealthCheck_numFail,
			)
			break
		case loadbalancer.StickyMethod_LBCookie:
			if req.StickySessionCookieName != "" {
				return nil, errors.New("sticky_session_cookie_name is not applicable for LBCookie sticky method")
			}
			if req.StickySessionTimeout <= 0 {
				return nil, errors.New("sticky_session_cookie_name must be greater than 0")
			}
			pool = loadbalancer.NewPoolWithStickySession(
				req.Hostname,
				time.Duration(req.HealthCheckTimeout*int64(time.Second)),
				time.Duration(req.HealthCheckInterval*int64(time.Second)),
				time.Duration(req.HealthCheckInitialDelay*int64(time.Second)),
				time.Duration(req.StickySessionTimeout*int64(time.Second)),
				req.HealthCheck_numOk,
				req.HealthCheck_numFail,
			)
		}
	} else {
		pool = loadbalancer.NewPool(
			req.Hostname,
			time.Duration(req.HealthCheckTimeout*int64(time.Second)),
			time.Duration(req.HealthCheckInterval*int64(time.Second)),
			time.Duration(req.HealthCheckInitialDelay*int64(time.Second)),
			req.HealthCheck_numOk,
			req.HealthCheck_numFail,
		)
	}
	return pool, nil
}
