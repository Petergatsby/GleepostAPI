package dir

import "sync"

type DirectoryList map[string]Directory2
type CachedList map[string]CachedDirectory

func (l DirectoryList) Register(dir Directory2, name string) {
	l[name] = dir
}

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
