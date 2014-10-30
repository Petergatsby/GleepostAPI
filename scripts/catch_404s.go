package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/draaglom/GleepostAPI/lib/conf"
)

func main() {
	conf := conf.GetConfig()
	database, err := sql.Open("mysql", conf.Mysql.ConnectionString())
	if err != nil {
		return
	}

	urlsStmt, err := database.Prepare("SELECT url FROM post_images")
	if err != nil {
		return
	}
	rows, err := urlsStmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var url string
		err := rows.Scan(&url)
		if err != nil {
			return
		}
		resp, err = http.Get(url)
		if err != nil {
			i++
		}
		log.Println(resp)

	}
	fmt.Println(i, "urls broke (should be 88)")
}
