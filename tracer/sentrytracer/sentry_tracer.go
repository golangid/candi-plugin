package sentrytracer

import (
	"context"
	"encoding/json"

	"github.com/getsentry/sentry-go"
	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/tracer"
)

// InitSentryTracer init sentry tracer
func InitSentryTracer(sentryTraceURL string) error {
	tracer.SetTracerPlatformType(&sentryPlatform{tracerHost: sentryTraceURL})
	return nil
}

type sentryPlatform struct {
	tracerHost string
}

func (s *sentryPlatform) StartSpan(ctx context.Context, operationName string) tracer.Tracer {
	span := sentry.StartSpan(ctx, operationName)
	return &sentryImpl{
		ctx:  span.Context(),
		span: span,
	}
}

func (s *sentryPlatform) StartRootSpan(ctx context.Context, operationName string, header map[string]string) tracer.Tracer {
	span := sentry.StartSpan(ctx, operationName, sentry.TransactionName(operationName))
	return &sentryImpl{
		ctx:  span.Context(),
		span: span,
	}
}
func (s *sentryPlatform) GetTraceID(ctx context.Context) (id string) {
	return
}
func (s *sentryPlatform) GetTraceURL(ctx context.Context) (u string) {
	return s.tracerHost
}

type sentryImpl struct {
	span *sentry.Span
	ctx  context.Context
	tags map[string]interface{}
}

func (s *sentryImpl) Context() context.Context {
	return s.ctx
}

func (s *sentryImpl) Tags() map[string]interface{} {
	if s.tags == nil {
		s.tags = make(map[string]interface{})
	}
	return s.tags
}

func (s *sentryImpl) SetTag(key string, value interface{}) {
	s.span.SetTag(key, string(candihelper.ToBytes(value)))
}

func (s *sentryImpl) InjectRequestHeader(header map[string]string) {
	header["sentry-trace"] = s.span.TraceID.String()
}

func (s *sentryImpl) SetError(err error) {
	if err != nil {
		s.span.Status = sentry.SpanStatusInternalError
		s.span.SetTag("error", err.Error())
	} else {
		s.span.Status = sentry.SpanStatusOK
	}
}

func (s *sentryImpl) Log(key string, value interface{}) {
	if s.span.Data == nil {
		s.span.Data = make(map[string]interface{})
	}

	switch v := value.(type) {
	case []byte:
		json.Unmarshal(v, &value)
	}
	s.span.Data[key] = value
}

func (s *sentryImpl) Finish(opt ...tracer.FinishOptionFunc) {
	for k, v := range s.tags {
		s.SetTag(k, v)
	}
	s.span.Finish()
}

// NewContext to continue tracer with new context
func (s *sentryImpl) NewContext() context.Context {
	return s.ctx
}
