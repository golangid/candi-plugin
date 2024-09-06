package oracledb_adaptor

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/sijms/go-ora/v2"
	"time"
)

var (
	oracleDriver = "oracle"
)

type oracleDBInstance struct {
	read, write *sql.DB
}

type OracleDatabaseOption func(db *sql.DB)

type OracleDatabase interface {
	ReadDB() *sql.DB
	WriteDB() *sql.DB
	Health() map[string]error
	Disconnect(ctx context.Context) error
}

func logWithDefer(str string) (deferFunc func()) {
	fmt.Printf("%s %s ", time.Now().Format(time.DateTime), str)
	return func() {
		if r := recover(); r != nil {
			fmt.Printf("\x1b[31;1mERROR: %v\x1b[0m\n", r)
			panic(r)
		}
		fmt.Println("\x1b[32;1mSUCCESS\x1b[0m")
	}
}

func (s *oracleDBInstance) ReadDB() *sql.DB {
	return s.read
}
func (s *oracleDBInstance) WriteDB() *sql.DB {
	return s.write
}

func (s *oracleDBInstance) Health() map[string]error {
	mErr := make(map[string]error)
	mErr["oracle_read"] = s.read.Ping()
	mErr["oracle_write"] = s.write.Ping()
	return mErr
}

// InitOracleDatabase return sql db read & write instance from environment:
// format dsn := oracle://user:password@host:port/db_name
// ORACLE_DB_READ_DSN, ORACLE_DB_WRITE_DSN
// if want to create single connection, use SQL_DB_WRITE_DSN and set empty for SQL_DB_READ_DSN
func InitOracleDatabase(readDsn, writeDsn string, opts ...OracleDatabaseOption) OracleDatabase {
	defer logWithDefer("Load OracleDB connection...")()

	if opts == nil {
		opts = append(opts, defaultOptions())
	}

	connReadDSN, connWriteDSN := readDsn, writeDsn
	if connReadDSN == "" {
		db := ConnectOracleDatabase(connWriteDSN, opts...)
		return &oracleDBInstance{
			read: db, write: db,
		}
	}

	return &oracleDBInstance{
		read:  ConnectOracleDatabase(connReadDSN, opts...),
		write: ConnectOracleDatabase(connWriteDSN, opts...),
	}
}

func (s *oracleDBInstance) Disconnect(ctx context.Context) (err error) {
	defer logWithDefer("\x1b[33;5moracledb\x1b[0m: disconnect...")()

	if err := s.read.Close(); err != nil {
		return err
	}
	return s.write.Close()
}

func ConnectOracleDatabase(dsn string, opts ...OracleDatabaseOption) *sql.DB {
	db, err := sql.Open(oracleDriver, dsn)
	if err != nil {
		panic(fmt.Sprintf("Oracle Connection: %v", err))
	}
	if err = db.Ping(); err != nil {
		panic(fmt.Sprintf("Oracle Ping: %v", err))
	}

	for _, opt := range opts {
		opt(db)
	}
	return db
}

func defaultOptions() (opt OracleDatabaseOption) {
	return func(db *sql.DB) {
		db.SetMaxIdleConns(5)
		db.SetMaxOpenConns(100)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Second)
	}
}
