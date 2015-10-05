//Package dir provides interfaces for looking up information in university directories.
package dir

//Directory represents a university student directory, in which you can look up people.
type Directory interface {
	LookUpEmail(email string) (userType, userID string, err error)
}

//NullDirectory is an empty directory. It can't look up anything.
type NullDirectory struct{}

//LookUpEmail will always return "student", nil
func (n NullDirectory) LookUpEmail(email string) (userType, userID string, err error) {
	return "student", "", nil
}

//TODO: add registry & function to return the appropriate directory by university name

//map[string]directory
//map[string]cachedDirectory

//directory interface:
//query(query, filter) -> []interface, err

//cachedDirectory interface:
//init(esUrl)
//index([]interface) -> err
//query(query, filter) -> []interface, err

//member interface
//->ID() -> string
//IsStudent() -> bool

//Directory2 is the interface common to all university directories.
type Directory2 interface {
	Query(query string, filter string) (results []interface{}, err error)
}

//CachedDirectory allows indexing directory results in a local elasticsearch cache.
type CachedDirectory interface {
	Init(esURL string)
	Index([]interface{}) (err error)
	Query(query string)
}

//Member is a Directory entry. ID should return a university-unique ID string for that person; Type() should attempt to indicate if the person is student, staff or faculty.
type Member interface {
	ID() string
	Type() string
}
