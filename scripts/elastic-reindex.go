package main

import (
	"github.com/Petergatsby/GleepostAPI/lib"
	"github.com/Petergatsby/GleepostAPI/lib/conf"
)

func main() {
	config := conf.GetConfig()
	api := lib.New(*config)

	api.ElasticSearchBulkReindex()
}
