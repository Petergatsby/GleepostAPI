package main

import (
	"github.com/draaglom/GleepostAPI/lib"
	"github.com/draaglom/GleepostAPI/lib/conf"
)

func main() {
	config := conf.GetConfig()
	api := lib.New(*config)

	api.ElasticSearchBulkReindex()
}
