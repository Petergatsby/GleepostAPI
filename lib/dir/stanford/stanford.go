package stanford

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mattbaird/elastigo/lib"
	"github.com/puerkitobio/goquery"
	"golang.org/x/net/html"
)

//Dir is stanford's directory
type Dir struct {
	ElasticSearch string
}

//LookUpEmail finds this user in the Stanford directory, and returns their type (staff, faculty, student)
func (d Dir) LookUpEmail(email string) (userType string, err error) {
	results, err := d.Query(email, Everyone)
	if err != nil {
		return
	}
	switch {
	case len(results) == 0:
		userType = "student"
	case len(results) > 1:
		userType = "student"
	case len(results[0].Affiliations) == 0:
		userType = "student"
	default:
		for _, aff := range results[0].Affiliations {
			if strings.Contains(aff.Affiliation, "Student") {
				userType = "student"
				break
			}
			if strings.Contains(aff.Affiliation, "Staff") {
				userType = "staff"
			}
			if strings.Contains(aff.Affiliation, "Faculty") {
				userType = "faculty"
			}
		}
	}
	return
}

//Member is a person in the Stanford directory.
type Member struct {
	Name          string        `json:"name"`
	ID            string        `json:"id"`
	Title         string        `json:"title,omitempty"`
	Email         string        `json:"email,omitempty"`
	Affiliations  []Affiliation `json:"affiliations"`
	MailCode      string        `json:"mail_code,omitempty"`
	HomeInfo      *HomeInfo     `json:"at_home,omitempty"`
	AlternateName string        `json:"other_name,omitempty"`
}

//Affiliation is (one of) a person's role(s) at Stanford.
type Affiliation struct {
	Affiliation string   `json:"name"`
	Department  string   `json:"department"`
	Position    string   `json:"position"`
	WorkPhones  []string `json:"phones,omitempty"`
	WorkFax     string   `json:"fax,omitempty"`
	WorkAddress string   `json:"address,omitempty"`
}

//HomeInfo is the information about a person's home life they've chosen to share in the stanford directory.
type HomeInfo struct {
	Phone   string `json:"phone,omitempty"`
	Address string `json:"address,omitempty"`
}

//Filters
//Everyone is the default filter, returning all results.
//University limits just to Stanford (ie, not Hospital / alums, but including med school)
//Faculty is just teaching staff
//Staff - regular staff, inc. eg cafeteria workers
//Student - undergrads, postgrads, etc
//Hospital - People associated with Stanford's hospital.
const (
	Everyone   = "everyone"
	University = "stanford:*"
	Faculty    = "stanford:faculty*"
	Staff      = "stanford:staff*"
	Student    = "stanford:student*"
	Hospital   = "sumc:staff*"
)

//Query searches the Stanford directory, performing a search against their website and scraping the results.
func (d Dir) Query(query string, filter string) (people []Member, err error) {
	c := &http.Client{}
	req, err := buildRequest(query, filter)
	if err != nil {
		return
	}
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	people, err = parseBody(resp)
	if err == nil {
		go d.bulkIndexMembers(people)
	}
	return
}

func buildRequest(query, filter string) (req *http.Request, err error) {
	searchURL := "https://stanfordwho.stanford.edu/SWApp/Search.do"
	if filter == "" {
		filter = Everyone
	}
	params := url.Values{}
	params.Set("search", query)
	params.Set("affilfilter", filter)

	body := params.Encode()
	r, err := http.NewRequest("POST", searchURL, bytes.NewBufferString(body))
	if err != nil {
		return
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return r, nil
}

//ErrParseFailure occurs when something unexpected happens trying to extract the results from the search markup.
var ErrParseFailure = errors.New("Parsing results page failed")

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
		return results, ErrParseFailure
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
	rawURL, exists := doc.Find("#ProfileNav ul li a").First().Attr("href")
	if exists {
		var URL *url.URL
		URL, err = url.Parse(rawURL)
		if err != nil {
			return
		}
		vals := URL.Query()
		result.ID = strings.TrimSpace(vals["key"][0])
	}
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
					vals := strings.Split(val, "-")
					for i, v := range vals {
						vals[i] = strings.TrimSpace(v)
					}
					aff.Affiliation = fmt.Sprintf("%s - %s", vals[0], vals[1])
				case lastLabel == "Department":
					aff.Department = val
				case lastLabel == "Position":
					aff.Position = val
				case lastLabel == "Work phone(s)":
					aff.WorkPhones = append(aff.WorkPhones, val)
				case lastLabel == "Work address":
					vals := strings.Split(val, "\n")
					for i, v := range vals {
						vals[i] = strings.TrimSpace(v)
					}
					aff.WorkAddress = strings.Join(vals, "\n")
				case lastLabel == "Work Fax":
					aff.WorkFax = val
				}
			}
		})
		result.Affiliations = append(result.Affiliations, aff)
	})
	var lastLabel string
	home := HomeInfo{}
	doc.Find("#HomeInfo dl").Children().Each(func(i int, s *goquery.Selection) {
		if s.Is("dt") {
			lastLabel = strings.TrimSpace(s.Text())
			lastLabel = lastLabel[:len(lastLabel)-1] //strip trailing colon
		} else if s.Is("dd") {
			val := strings.TrimSpace(s.Text())
			switch {
			case lastLabel == "Permanent phone":
				home.Phone = val
			case lastLabel == "Permanent address":
				vals := strings.Split(val, "\n")
				for i, v := range vals {
					vals[i] = strings.TrimSpace(v)
				}
				home.Address = strings.Join(vals, "\n")
			}
		}
		result.HomeInfo = &home
	})
	doc.Find("#Ids dl").Children().Each(func(i int, s *goquery.Selection) {
		if s.Is("dt") {
			lastLabel = strings.TrimSpace(s.Text())
			lastLabel = lastLabel[:len(lastLabel)-1] //strip trailing colon
		} else if s.Is("dd") {
			val := strings.TrimSpace(s.Text())
			switch {
			case lastLabel == "Other names":
				result.AlternateName = val
			case lastLabel == "Inter-dept mail code":
				result.MailCode = val
			}
		}
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
				member.ID = strings.TrimSpace(vals["key"][0])
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

//CacheQuery searches the local elasticsearch cache of the Stanford directory.
func (d Dir) CacheQuery(query, filter string) (people []Member, err error) {
	people = make([]Member, 0)
	c := elastigo.NewConn()
	c.Domain = d.ElasticSearch
	if filter == "" {
		filter = Everyone
	}

	esQuery := composeQuery(query, filter)

	results, err := c.Search("directory", "stanford", nil, esQuery)
	if err != nil {
		return
	}
	for _, hit := range results.Hits.Hits {
		var member Member
		err = json.Unmarshal(*hit.Source, &member)
		if err != nil {
			return
		}
		people = append(people, member)
	}
	return
}

func composeQuery(query, filter string) (esQuery interface{}) {
	fields := []string{"name", "name.partial", "name.metaphone", "other_name", "other_name.partial", "other_name.metaphone"}
	if filter == Everyone {
		q := esqueryNoFilter{}
		for _, field := range fields {
			match := make(map[string]string)
			matcher := matcher{Match: match}
			matcher.Match[field] = query
			q.Query.Bool.Should = append(q.Query.Bool.Should, matcher)
		}
		return q
	} else {
		q := esquery{}
		term := make(map[string]string)
		term["affiliations.name"] = filter
		q.Query.Filtered.Filter.Term = term
		for _, field := range fields {
			match := make(map[string]string)
			matcher := matcher{Match: match}
			matcher.Match[field] = query
			q.Query.Filtered.Query.Bool.Should = append(q.Query.Filtered.Query.Bool.Should, matcher)
		}
		return q
	}
}

type esquery struct {
	Query innerquery `json:"query"`
}

type innerquery struct {
	Filtered filteredquery `json:"filtered"`
}

type filteredquery struct {
	Filter filter          `json:"filter"`
	Query  innerinnerquery `json:"query"`
}

type innerinnerquery struct {
	Bool boolquery `json:"bool"`
}

type filter struct {
	Term map[string]string `json:"term"`
}

type boolquery struct {
	Should []matcher `json:"should"`
}

type matcher struct {
	Match map[string]string `json:"match"`
}

type esqueryNoFilter struct {
	Query innerinnerquery `json:"query"`
}

func (d Dir) bulkIndexMembers(members []Member) {
	c := elastigo.NewConn()
	c.Domain = d.ElasticSearch
	indexer := c.NewBulkIndexerErrors(10, 60)
	indexer.Start()
	defer indexer.Stop()
	for _, member := range members {
		indexer.Index("directory", "stanford", member.ID, "", nil, member, true)
	}
}
