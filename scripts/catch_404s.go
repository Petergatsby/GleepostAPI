package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/draaglom/GleepostAPI/lib/conf"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	conf := conf.GetConfig()
	database, err := sql.Open("mysql", conf.Mysql.ConnectionString())
	if err != nil {
		log.Fatal(err)
	}

	urlsStmt, err := database.Prepare("SELECT url FROM post_images")
	if err != nil {
		log.Fatal(err)
	}
	rows, err := urlsStmt.Query()
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var url string
		err := rows.Scan(&url)
		if err != nil {
			return
		}
		log.Println(url)
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode == 403 {
			i++
			fmt.Println(url)
		}
	}
	fmt.Println(i, "urls broke (should be 88)")
}
