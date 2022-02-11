package arangodb_adapter

import (
	"context"
	"fmt"
	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"log"
)

type arangoInstance struct {
	read, write driver.Database
}

func (a *arangoInstance) ReadDB() driver.Database {
	return a.read
}

func (a *arangoInstance) WriteDB() driver.Database {
	return a.write
}

func (m *arangoInstance) Health() map[string]error {
	return map[string]error{
		"arango_read": nil, "arango_write": nil,
	}
}

func (m *arangoInstance) Disconnect(ctx context.Context) (err error) {
	return
}

// InitArangoDB return mongo db read & write instance from environment:
// ARANGODB_HOST_WRITE, ARANGODB_HOST_READ
func InitArangoDB(ctx context.Context, env ArangoDBEnv) ArangoDatabase {
	log.Print("Load ArangoDB connection...")
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("\x1b[31;1mERROR: %v\x1b[0m\n", rec)
			panic(rec)
		}
		log.Print("\x1b[32;1mSUCCESS\x1b[0m")
	}()

	readAuthentication := driver.BasicAuthentication(env.DbArangoReadUser, env.DbArangoReadPassword)
	writeAuthentication := driver.BasicAuthentication(env.DbArangoWriteUser, env.DbArangoWritePassword)
	return &arangoInstance{
		read:  ConnectArangoDB(ctx, env.DbArangoReadHost, env.DbArangoReadDatabase, readAuthentication),
		write: ConnectArangoDB(ctx, env.DbArangoWriteHost, env.DbArangoWriteDatabase, writeAuthentication),
	}
}

// ConnectArangoDB connect to mongodb with dsn
func ConnectArangoDB(ctx context.Context, host string, dbName string, authentication driver.Authentication) driver.Database {
	connection, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{host},
	})
	if err != nil {
		panic(fmt.Errorf("arangodb error connect: %v", err))
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     connection,
		Authentication: authentication,
	})
	if err != nil {
		panic(err)
	}

	db, err := client.Database(ctx, dbName)
	if err != nil {
		panic(err)
	}

	return db
}
