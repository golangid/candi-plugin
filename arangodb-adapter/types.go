package arangodb_adapter

type ArangoDBEnv struct {
	DbArangoWriteHost, DbArangoReadHost         string
	DbArangoWriteUser, DbArangoReadUser         string
	DbArangoWritePassword, DbArangoReadPassword string
	DbArangoReadDatabase, DbArangoWriteDatabase string
}
