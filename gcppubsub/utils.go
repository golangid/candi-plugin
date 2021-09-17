package gcppubsub

import (
	"context"

	"github.com/golangid/candi/candishared"
)

// GetMessageAttributes get message attributes from context
func GetMessageAttributes(ctx context.Context) map[string]string {
	return candishared.GetValueFromContext(ctx, MessageAttribute).(map[string]string)
}
