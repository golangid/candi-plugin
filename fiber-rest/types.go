package fiberrest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golangid/candi/codebase/factory/types"
)

const (
	// FiberREST types
	FiberREST types.Server = "fiber-rest"
)

// ParseGroupHandler parse mount handler param to fiber group
func ParseGroupHandler(i interface{}) *fiber.Group {
	return i.(*fiber.Group)
}
