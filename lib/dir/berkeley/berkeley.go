package berkeley

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/puerkitobio/goquery"
)

//Dir is berkeley's directory
type Dir struct{}

//LookUpEmail finds this user in the Berkeley directory, and returns their type (staff, faculty, student)
//Pretty inaccurate, but good as a first pass.
func (d Dir) LookUpEmail(email string) (userType string, err error) {
	c := &http.Client{}
	searchURL := "http://www.berkeley.edu/directory/results?search-term="
	searchURL += url.QueryEscape(email)
	resp, err := c.Get(searchURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}
	doc.Find(".search-results p").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Title") {
			userType = "staff"
		}
		if strings.Contains(s.Text(), "Prof") {
			userType = "faculty"
		}
	})
	if userType == "" {
		userType = "student"
	}
	return

}
