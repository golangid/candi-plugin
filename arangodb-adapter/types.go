package arangodb_adapter

import (
	"net/url"
	"strings"
)

type ArangoDBEnv struct {
	DbArangoWriteHost, DbArangoReadHost         string
	DbArangoWriteUser, DbArangoReadUser         string
	DbArangoWritePassword, DbArangoReadPassword string
	DbArangoReadDatabase, DbArangoWriteDatabase string
}

var arangoDBEnv ArangoDBEnv

func parseArangoEnv(dbReadDSN, dbWriteDSN string) ArangoDBEnv {
	read, err := url.Parse(dbReadDSN)
	if err != nil {
		panic(err)
	}

	arangoDBEnv.DbArangoReadHost = read.Scheme + "://" + read.Host
	arangoDBEnv.DbArangoReadUser = read.User.Username()
	arangoDBEnv.DbArangoReadPassword, _ = read.User.Password()
	arangoDBEnv.DbArangoReadDatabase = strings.Trim(read.Path, "/")

	write, err := url.Parse(dbWriteDSN)
	if err != nil {
		panic(err)
	}
	arangoDBEnv.DbArangoWriteHost = write.Scheme + "://" + read.Host
	arangoDBEnv.DbArangoWriteUser = write.User.Username()
	arangoDBEnv.DbArangoWritePassword, _ = write.User.Password()
	arangoDBEnv.DbArangoWriteDatabase = strings.Trim(write.Path, "/")

	return arangoDBEnv
}