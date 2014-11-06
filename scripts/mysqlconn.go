package main

import (
	"fmt"

	"github.com/draaglom/GleepostAPI/lib/conf"
)

func main() {
	conf := conf.GetConfig()
	fmt.Println(conf.Mysql.ConnectionString())

}
