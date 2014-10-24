package main

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/puerkitobio/goquery"
)

func main() {
	//Connect to the db
	conf := conf.GetConfig()
	database := db.New(conf.Mysql)

	emails, err := database.AllEmails()
	if err != nil {
		log.Println(err)
		return
	}
	staff := 0
	total := 0
	for _, email := range emails {
		teacher, err := lookUp(email)
		if err != nil {
			log.Println(err)
		}
		if teacher {
			staff++
		}
		total++
	}
	log.Println("Staff:", staff, "Total:", total)
}

func lookUp(email string) (isStaff bool, err error) {
	c := &http.Client{}
	searchURL := "https://stanfordwho.stanford.edu/SWApp/Search.do"
	params := url.Values{}
	params.Set("search", email)
	body := params.Encode()
	r, err := http.NewRequest("POST", searchURL, bytes.NewBufferString(body))
	if err != nil {
		return
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(r)
	log.Println(resp.Status)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}
	doc.Find(".affilHead").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Staff") {
			isStaff = true
		}
		if strings.Contains(s.Text(), "Faculty") {
			isStaff = true
		}
	})
	return isStaff, nil
}
