package psc

import "database/sql"

//StatementCache caches prepared statements.
type StatementCache struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
}

//Prepare gives you an already prepared statement if available, otherwise prepares one with the underlying sql.db.
func (s *StatementCache) Prepare(query string) (stmt *sql.Stmt, err error) {
	stmt, ok := s.stmts[query]
	if ok {
		return stmt, nil
	}
	stmt, err = s.db.Prepare(query)
	if err != nil {
		return
	}
	s.stmts[query] = stmt
	return stmt, nil
}

//NewCache creates a new prepared statement cache.
func NewCache(db *sql.DB) (s *StatementCache) {
	s = &StatementCache{db: db}
	s.stmts = make(map[string]*sql.Stmt)
	return
}
