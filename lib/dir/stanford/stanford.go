package stanford

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/puerkitobio/goquery"
	"golang.org/x/net/html"
)

//Dir is stanford's directory
type Dir struct{}

//LookUpEmail finds this user in the Stanford directory, and returns their type (staff, faculty, student)
func (d Dir) LookUpEmail(email string) (userType string, err error) {
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

type Member struct {
	Name         string
	Title        string
	Email        string
	Affiliations []Affiliation
	ID           string
	MailCode     string
	//Other info:
	//Home address
	//Other names
}

type Affiliation struct {
	Affiliation string
	Department  string
	Position    string
	WorkPhones  []string
	//WorkFax string
	WorkAddress string
}

func (d Dir) Query(query string) (people []Member, err error) {
	c := &http.Client{}
	req, err := buildRequest(query)
	if err != nil {
		return
	}
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	people, err = parseBody(resp)
	return
}

func buildRequest(query string) (req *http.Request, err error) {
	searchURL := "https://stanfordwho.stanford.edu/SWApp/Search.do"
	params := url.Values{}
	params.Set("search", query)
	body := params.Encode()
	r, err := http.NewRequest("POST", searchURL, bytes.NewBufferString(body))
	if err != nil {
		return
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return r, nil
}

var ParseFailure = errors.New("Parsing results page failed")

func parseBody(resp *http.Response) (results []Member, err error) {
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}
	//Determine if we have a single user or a results list
	individual := false
	heading := doc.Find("#ResultsHead h1").First().Text()
	switch {
	case strings.Contains(heading, "No matches"):
		return results, nil
	case strings.Contains(heading, "Public listing"):
		individual = true
	case strings.Contains(heading, "matches in public directory"):
		individual = false
	default:
		return results, ParseFailure
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
	result.Name = strings.TrimSpace(doc.Find("#PublicProfile h2").First().Text())
	result.Title = strings.TrimSpace(doc.Find("#PublicProfile p.facappt").First().Text())
	result.Email = strings.TrimSpace(doc.Find("#Contact dl dd a").First().Text())
	result.MailCode = strings.TrimSpace(doc.Find("#Ids dl dd").First().Text())
	//Todo: student ID
	doc.Find(".Affiliation").Each(func(i int, s *goquery.Selection) {
		aff := Affiliation{}
		var lastLabel string
		s.Find("dl").Children().Each(func(i int, s *goquery.Selection) {
			if s.Is("dt") {
				lastLabel = strings.TrimSpace(s.Text())
				lastLabel = lastLabel[:len(lastLabel)-1] //strip trailing colon
			} else if s.Is("dd") {
				val := strings.TrimSpace(s.Text())
				switch {
				case lastLabel == "Affiliation":
					aff.Affiliation = val
				case lastLabel == "Department":
					aff.Department = val
				case lastLabel == "Position":
					aff.Position = val
				case lastLabel == "Work phone(s)":
					aff.WorkPhones = append(aff.WorkPhones, val)
				case lastLabel == "Work address":
					aff.WorkAddress = val
				}
			}
		})
		result.Affiliations = append(result.Affiliations, aff)
	})
	return result, nil
}

func parseMultipleResults(doc *goquery.Document) (results []Member, err error) {
	var member Member
	doc.Find("#PublicResults dl").Children().Each(func(i int, s *goquery.Selection) {
		if s.Is("dt") {
			member = Member{}
			member.Name = strings.TrimSpace(s.Find("a").First().Text())
			rawurl, exists := s.Find("a").First().Attr("href")
			if exists {
				URL, err := url.Parse(rawurl)
				if err != nil {
					return
				}
				vals := URL.Query()
				member.ID = vals["key"][0]
			}

		} else if s.Is("dd") {
			rawAffils := strings.TrimSpace(s.Find("ul li span.affil").First().Text())
			if len(rawAffils) > 0 {
				affils := parseAffils(rawAffils)
				var position string
				for _, node := range s.Find("ul li").Contents().Nodes {
					if node.Type == html.TextNode {
						if position == "" {
							position = strings.TrimSpace(node.Data)
						} else {
							//Number of full affil names might not match
							//the number of node-pairs we have here.
							//That's because the results collapse duplicate titles:
							//eg (University - staff, staff, student) will become
							//(University - staff, student).
							//Because of that we can't really be sure
							//in these cases which node-pairs here map to which of the
							//smaller set of affils.
							//eg https://stanfordwho.stanford.edu/SWApp/detailAction.do?key=DR967E786&search=patrick&soundex=&stanfordonly=&affilfilter=everyone&filters=closed
							aff := Affiliation{
								Affiliation: affils[0],
								Department:  strings.TrimSpace(node.Data),
								Position:    position,
							}
							position = ""
							member.Affiliations = append(member.Affiliations, aff)
						}
					}
				}
				if len(affils) > 2 {
					panic("!!! Found one !!!")
				}
				//So we just fudge it and say only the last affil will have the last affil name.
				if len(affils) > 1 && len(member.Affiliations) >= len(affils) {
					member.Affiliations[len(member.Affiliations)-1].Affiliation = affils[len(affils)-1]
				}
			}
			results = append(results, member)
		}
	})
	return results, nil
}

func parseAffils(combinedAffils string) (affils []string) {
	//in the results list, affiliations are enclosed in parens
	combinedAffils = combinedAffils[1 : len(combinedAffils)-1]
	//Assuming that there will only be one - separator...
	splitAffils := strings.SplitN(combinedAffils, "-", 2)
	//Also assuming there will only be commas if the person holds multiple roles (and not in the role names)
	splitPos := strings.Split(splitAffils[1], ",")
	institution := strings.TrimSpace(splitAffils[0])
	for _, pos := range splitPos {
		pos = strings.TrimSpace(pos)
		affils = append(affils, fmt.Sprintf("%s - %s", institution, pos))
	}
	return
}
