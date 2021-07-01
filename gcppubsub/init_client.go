package gcppubsub

import (
	"context"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

// InitDefaultClient gcp pubsub
func InitDefaultClient(gcpProjectName, credentialPath string) *pubsub.Client {
	client, err := pubsub.NewClient(context.Background(), gcpProjectName, option.WithCredentialsFile(credentialPath))
	if err != nil {
		panic(err)
	}

	return client
}
