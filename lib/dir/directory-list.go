package dir

import "sync"

//DirectoryList contains all the registered university-specific directories.
type DirectoryList map[string]Directory2

//CachedList contains all the registered locally-cached university directories.
type CachedList map[string]CachedDirectory

//Register makes a Directory available in the DirectoryList; it should be called in an implementor package's init().
func (l DirectoryList) Register(dir Directory2, name string) {
	l[name] = dir
}

//RegisterCache makes a CachedDirectory available in the CachedList; it should be called in an implementor package's init().
func (l CachedList) RegisterCache(dir CachedDirectory, name string) {
	l[name] = dir
}

var directories DirectoryList
var cached CachedList
var dm = &sync.Mutex{}
var cm = &sync.Mutex{}

func init() {
	directories = make(map[string]Directory2)
	cached = make(map[string]CachedDirectory)
}
