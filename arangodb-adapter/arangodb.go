package arangodb_adapter

import (
	"context"
	"fmt"
	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"time"
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
func InitArangoDB(ctx context.Context, dsnRead string, dsnWrite string) ArangoDatabase {
	fmt.Printf("%s Load ArangoDB connection... ", time.Now().Format("2006/01/02 15:04:05"))
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("\x1b[31;1mERROR: %v\x1b[0m\n", rec)
			panic(rec)
		}
		fmt.Println("\x1b[32;1mSUCCESS\x1b[0m")
	}()

	env := parseArangoEnv(dsnRead, dsnWrite)
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
