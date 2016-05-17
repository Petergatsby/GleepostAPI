package main

import (
	"fmt"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
)

func main() {
	conf := conf.GetConfig()
	fmt.Println(conf.Mysql.ConnectionString())

}
