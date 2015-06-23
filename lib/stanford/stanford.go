package stanford

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"

	"github.com/puerkitobio/goquery"
)

//LookUp finds this user in the Stanford directory, and returns their type (staff, faculty, student)
func LookUp(email string) (userType string, err error) {
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
			userType = "staff"
		}
		if strings.Contains(s.Text(), "Faculty") {
			userType = "faculty"
		}
	})
	if userType == "" {
		userType = "student"
	}
	return
}
