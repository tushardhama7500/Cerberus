package resolver

import (
	"cerberus/internal/service"
)

type Resolver struct {
	AuthService          *service.AuthService
	AccessRequestService *service.AccessRequestService
}
