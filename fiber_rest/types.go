package fiberrest

import (
	"github.com/gofiber/fiber/v2"
	"pkg.agungdp.dev/candi/codebase/factory/types"
)

const (
	// FiberREST types
	FiberREST types.Server = "fiber_rest"
)

// ParseGroupHandler parse mount handler param to fiber group
func ParseGroupHandler(i interface{}) *fiber.Group {
	return i.(*fiber.Group)
}
