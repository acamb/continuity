package responses

import (
	"continuity/common"
	"continuity/server/loadbalancer"
	"net/url"

	"github.com/google/uuid"
)

type ServerHostResponse struct {
	Id              uuid.UUID
	Address         *url.URL
	Condition       common.Condition
	ServerStatus    string
	HealthCheckPath string
	createdAt       int64
}

func NewServerHostResponse(server *loadbalancer.ServerHost) *ServerHostResponse {
	return &ServerHostResponse{
		Id:              server.Id,
		Address:         server.Address,
		Condition:       server.Condition,
		ServerStatus:    loadbalancer.ServerStatus(server.ServerStatus.Load()).String(),
		HealthCheckPath: server.HealthCheckPath,
		createdAt:       server.CreatedAt,
	}
}

func (shr *ServerHostResponse) String() string {
	return "Server " + shr.Id.String() + ":\n" +
		"\t\t\tAddress: " + shr.Address.String() + "\n" +
		"\t\t\tCondition: " + shr.Condition.String() + "\n" +
		"\t\t\tServerStatus: " + shr.ServerStatus + "\n" +
		"\t\t\tHealthCheckPath: " + shr.HealthCheckPath + "\n"
}
