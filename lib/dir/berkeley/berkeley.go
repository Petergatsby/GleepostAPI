package berkeley

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/puerkitobio/goquery"
	"golang.org/x/net/html"
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

//Query searches the Berkeley directory.
func (d Dir) Query(query string) (results []Member, err error) {
	c := &http.Client{}
	req, err := buildRequest(query)
	if err != nil {
		return
	}
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	results, err = parseBody(resp)
	return

}

func buildRequest(query string) (req *http.Request, err error) {
	searchURL := "http://www.berkeley.edu/directory/results?search-term="
	searchURL += url.QueryEscape(query)
	req, err = http.NewRequest("GET", searchURL, nil)
	return
}

func parseBody(resp *http.Response) (results []Member, err error) {
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}
	//Determine if we have a single user or a results list
	individual := true
	heading := doc.Find(".search-results p").Text()
	switch {
	case strings.Contains(heading, "No matches"):
		return results, nil
	case strings.Contains(heading, "people found"):
		individual = false
	case strings.Contains(heading, "people matching"):
		individual = false
	}
	if individual {
		var member Member
		member, err = parseIndividualResult(doc)
		results = append(results, member)
	} else {
		results, err = parseMultipleResults(doc)
	}
	return
}

func parseIndividualResult(doc *goquery.Document) (result Member, err error) {
	result.Name = doc.Find(".search-results h2").Text()
	result.Email = doc.Find(".search-results p a").Text()
	var lastLabel string
	var addressLines []string
	for _, node := range doc.Find(".search-results p").Contents().Nodes {
		switch {
		case node.Type == html.ElementNode && node.Data == "label" && node.FirstChild != nil:
			lastLabel = node.FirstChild.Data
			switch {
			case len(result.Address) == 0 && len(addressLines) > 0:
				result.Address = strings.Join(addressLines, "\n")
				addressLines = []string{}
			case len(result.Address) > 0 && len(addressLines) > 0:
				result.AddressAdditional = strings.Join(addressLines, "\n")
				addressLines = []string{}
			}
		case node.Type == html.TextNode && (lastLabel == "Home department"):
			result.HomeDepartment = node.Data
		case node.Type == html.TextNode && (lastLabel == "UID"):
			result.ID = node.Data
		case node.Type == html.TextNode && (lastLabel == "Title"):
			result.Title = node.Data
		case node.Type == html.TextNode && (lastLabel == "Department"):
			result.Department = node.Data
		case node.Type == html.TextNode && (lastLabel == "Address"):
			newLines := strings.Split(strings.TrimSpace(node.Data), "\n")
			for _, line := range newLines {
				addressLines = append(addressLines, strings.TrimSpace(line))
			}
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil && lastLabel == "Email":
			result.Email = node.FirstChild.Data
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil && lastLabel == "Website":
			result.Website = node.FirstChild.Data
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil && lastLabel == "Telephone" && result.Telephone == "":
			result.Telephone = node.FirstChild.Data
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil && lastLabel == "Telephone":
			result.TelephoneAdditional = node.FirstChild.Data
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil && lastLabel == "Fax":
			result.Fax = node.FirstChild.Data
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil && lastLabel == "Mobile":
			result.Mobile = node.FirstChild.Data
		case node.Type == html.ElementNode && node.Data == "a" && node.FirstChild != nil:
			log.Println(lastLabel, node.FirstChild.Data)
		case node.Type == html.TextNode:
			log.Println(lastLabel, strings.TrimSpace(node.Data))
		}
	}
	return
}

type Member struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Email               string `json:"email,omitempty"`
	HomeDepartment      string `json:"home_department,omitempty"`
	Title               string `json:"title,omitempty"`
	Department          string `json:"department,omitempty"`
	Website             string `json:"website,omitempty"`
	Address             string `json:"address,omitempty"`
	Telephone           string `json:"phone,omitempty"`
	Fax                 string `json:"fax,omitempty"`
	AddressAdditional   string `json:"address_additional,omitempty"`
	TelephoneAdditional string `json:"phone_additional,omitempty"`
	Mobile              string `json:"mobile,omitempty"`
}

func parseMultipleResults(doc *goquery.Document) (results []Member, err error) {
	results = make([]Member, 0)
	doc.Find(".search-results ul li a").Each(func(i int, s *goquery.Selection) {
		result := Member{}
		result.Name = normalizeListName(s.Text())

		rawURL, exists := s.Attr("href")
		if exists {
			var URL *url.URL
			URL, err = url.Parse(rawURL)
			if err != nil {
				return
			}
			vals := URL.Query()
			result.ID = strings.TrimSpace(vals["u"][0])
		}
		results = append(results, result)
	})
	return
}

func normalizeListName(name string) (normalized string) {
	split := strings.Split(name, ",")
	if len(split) > 1 {
		normalized = strings.Title(strings.ToLower(split[1])) + " " + strings.Title(strings.ToLower(split[0]))
	}
	if len(split) > 2 {
		//Handles eg. Smith, John, Jr -> John Smith, Jr
		//Not mapping to title case because it could be "Smith, John, IV" -> "John Smith, IV"
		normalized = normalized + ", " + split[2]
	}
	return normalized
}
