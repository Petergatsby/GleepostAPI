package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
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
	fixStmt, err := database.Prepare("UPDATE post_images SET url = CONCAT('http://d2tc2ce3464r63.cloudfront.net/', SUBSTRING(url, 38)) WHERE url = ?")
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
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode == 403 {
			fmt.Println(url)
			_, err = fixStmt.Exec(url)
			if err != nil {
				log.Println(err)
			} else {
				i++
			}

		}
	}
	fmt.Println(i, "urls fixed (should be 88)")
}
