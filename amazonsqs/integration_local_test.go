//go:build integration

package amazonsqs

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/dependency"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	cfgPkg "github.com/golangid/candi/config"
)

const elasticMQEndpoint = "http://localhost:9324"

func newLocalClient(t *testing.T) *sqs.Client {
	t.Helper()
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("x", "x", "")),
		awsconfig.WithBaseEndpoint(elasticMQEndpoint),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return sqs.NewFromConfig(cfg)
}

// --- minimal fakes implementing the candi factory contracts ---

type fakeHandler struct{ mounter func(*types.WorkerHandlerGroup) }

func (h fakeHandler) MountHandlers(g *types.WorkerHandlerGroup) { h.mounter(g) }

type fakeModule struct{ wh interfaces.WorkerHandler }

func (m fakeModule) RESTHandler() interfaces.RESTHandler       { return nil }
func (m fakeModule) GRPCHandler() interfaces.GRPCHandler       { return nil }
func (m fakeModule) GraphQLHandler() interfaces.GraphQLHandler { return nil }
func (m fakeModule) WorkerHandler(_ types.Worker) interfaces.WorkerHandler {
	return m.wh
}
func (m fakeModule) ServerHandler(_ types.Server) interfaces.ServerHandler { return nil }
func (m fakeModule) Name() types.Module                                    { return "test" }

type fakeService struct {
	modules []factory.ModuleFactory
	deps    dependency.Dependency
}

func (s fakeService) GetConfig() *cfgPkg.Config              { return nil }
func (s fakeService) GetDependency() dependency.Dependency   { return s.deps }
func (s fakeService) GetApplications() []factory.AppServerFactory { return nil }
func (s fakeService) GetModules() []factory.ModuleFactory    { return s.modules }
func (s fakeService) Name() types.Service                    { return "test-service" }

func TestPublishAndConsume_Local(t *testing.T) {
	client := newLocalClient(t)
	ctx := context.Background()

	const queueName = "candi-sqs-e2e"
	if _, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(queueName)}); err != nil {
		t.Fatalf("create queue: %v", err)
	}

	broker := NewAmazonSQSBroker(client)

	// publish a message addressed by queue NAME (publisher resolves to url)
	const want = `{"hello":"world"}`
	if err := broker.GetPublisher().PublishMessage(ctx, &candishared.PublisherArgument{
		Topic:   queueName,
		Message: []byte(want),
		Header:  map[string]interface{}{"x-custom": "abc"},
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// wire a consumer handler that captures the consumed message
	var (
		mu       sync.Mutex
		got      string
		gotHdr   string
		consumed = make(chan struct{}, 1)
	)
	handler := fakeHandler{mounter: func(g *types.WorkerHandlerGroup) {
		g.Add(queueName, func(ec *candishared.EventContext) error {
			mu.Lock()
			got = string(ec.Message())
			gotHdr = ec.Header()["x-custom"]
			mu.Unlock()
			select {
			case consumed <- struct{}{}:
			default:
			}
			return nil
		})
	}}

	deps := dependency.InitDependency(dependency.SetValidator(nil))
	svc := fakeService{modules: []factory.ModuleFactory{fakeModule{wh: handler}}, deps: deps}

	worker := NewAmazonSQSWorker(svc, broker, 10, 1)
	go worker.Serve()
	defer worker.Shutdown(context.Background())

	select {
	case <-consumed:
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for message to be consumed")
	}

	mu.Lock()
	defer mu.Unlock()
	if got != want {
		t.Fatalf("consumed body mismatch: got %q want %q", got, want)
	}
	if gotHdr != "abc" {
		t.Fatalf("custom header not propagated: got %q want %q", gotHdr, "abc")
	}

	// give AutoACK a moment, then confirm the queue is drained (message deleted)
	time.Sleep(1 * time.Second)
	attrs, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       func() *string { u, _ := broker.resolveQueueURL(ctx, queueName); return aws.String(u) }(),
		AttributeNames: []sqstypes.QueueAttributeName{"ApproximateNumberOfMessages"},
	})
	if err != nil {
		t.Fatalf("get queue attrs: %v", err)
	}
	if n := attrs.Attributes["ApproximateNumberOfMessages"]; n != "0" {
		t.Fatalf("expected queue drained after AutoACK, ApproximateNumberOfMessages=%s", n)
	}
}
