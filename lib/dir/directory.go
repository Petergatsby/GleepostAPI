//Package dir provides interfaces for looking up information in university directories.
package dir

//Directory represents a university student directory, in which you can look up people.
type Directory interface {
	LookUpEmail(email string) (userType string, err error)
}

//NullDirectory is an empty directory. It can't look up anything.
type NullDirectory struct{}

//LookUpEmail will always return "student", nil
func (n NullDirectory) LookUpEmail(email string) (userType string, err error) {
	return "student", nil
}
