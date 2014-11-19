//Package db contains everything to do with accessing the database.
//it's dependent on mysql-specific features (REPLACE INTO).
//As well as a prepared statement cache which arose more or less accidentally, but which will be useful for stats later.
package db

import (
	"database/sql"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/gp"
)

const (
	//For parsing
	mysqlTime = "2006-01-02 15:04:05"
)

var (
	//UserAlreadyExists appens when creating an account with a dupe email address.
	UserAlreadyExists = gp.APIerror{Reason: "Username or email address already taken"}
)

//DB contains the database configuration and so forth.
type DB struct {
	stmt     map[string]*sql.Stmt
	database *sql.DB
	config   conf.MysqlConfig
}

//New creates a DB; it connects an underlying sql.db and will fatalf if it can't.
func New(conf conf.MysqlConfig) (db *DB) {
	var err error
	db = new(DB)
	db.database, err = sql.Open("mysql", conf.ConnectionString())
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.database.SetMaxIdleConns(conf.MaxConns)
	db.stmt = make(map[string]*sql.Stmt)
	return db
}

//prepare wraps sql.DB.Prepare, storing prepared statements in a map.
func (db *DB) prepare(statement string) (stmt *sql.Stmt, err error) {
	stmt, ok := db.stmt[statement]
	if ok {
		return
	}
	stmt, err = db.database.Prepare(statement)
	if err == nil {
		db.stmt[statement] = stmt
	}
	return
}
