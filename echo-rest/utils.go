package echorest

import (
	"context"
	"net/http"

	"github.com/golangid/candi/candishared"
	graphqlserver "github.com/golangid/candi/codebase/app/graphql_server"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/config/env"
	"github.com/labstack/echo"
)

var (
	echoURLParamCtxKey = candishared.ContextKey("echoURLParamCtxKey")
)

func setToContext(ctx context.Context, value map[string]string) context.Context {
	return context.WithValue(ctx, echoURLParamCtxKey, value)
}

func getRouteContext(ctx context.Context) map[string]string {
	val, _ := ctx.Value(echoURLParamCtxKey).(map[string]string)
	return val
}

// ParseRouter parse mount handler param to echo group router
func ParseRouter(i interface{}) *echo.Group {
	return i.(*echo.Group)
}

// URLParam get url param from request context
func URLParam(req *http.Request, key string) string {
	return getRouteContext(req.Context())[key]
}

// SetupRESTServer setup rest server with default config
func SetupRESTServer(service factory.ServiceFactory, opts ...OptionFunc) factory.AppServerFactory {
	restOptions := []OptionFunc{
		SetHTTPPort(env.BaseEnv().HTTPPort),
		SetRootPath(env.BaseEnv().HTTPRootPath),
		SetIncludeGraphQL(env.BaseEnv().UseGraphQL),
		SetSharedListener(service.GetConfig().SharedListener),
		SetDebugMode(env.BaseEnv().DebugMode),
		SetJaegerMaxPacketSize(env.BaseEnv().JaegerMaxPacketSize),
	}
	if env.BaseEnv().UseGraphQL {
		restOptions = append(restOptions, AddGraphQLOption(
			graphqlserver.SetDisableIntrospection(env.BaseEnv().GraphQLDisableIntrospection),
			graphqlserver.SetHTTPPort(env.BaseEnv().HTTPPort),
		))
	}
	restOptions = append(restOptions, opts...)
	return NewServer(service, restOptions...)
}
