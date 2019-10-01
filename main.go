package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	mw "bitbucket.org/iharsuvorau/mediawiki"
)

var (
	// errResponseParsing signifies the type conversion error. Usually, it
	// means there is unexpected content in the response.
	errResponseParsing = errors.New("unexpected response")

	// errNotFound signifies not found content.
	errNotFound = errors.New("not found")

	// regular expressions for templates parsing

	// capturing groups: username, name, description
	reTeamMember = regexp.MustCompile(`{{\s*TeamMember\s*\|\s*(.*)\s*\|\s*(.*)\s*\|\s*(.*)}}\s*`)
	// capturing groups: username, name, description, thesis
	reAlumnus = regexp.MustCompile(`{{\s*Alumnus\s*\|\s*(.*)\s*\|\s*(.*)\s*\|\s*(.*)\s*\|\s*(.*)}}\s*`)
)

type user struct {
	FullName, UserName, Description string
	Theses                          []*thesis

	// ThesesString is used to read the markup from Alumni page without
	// changing its content.
	ThesesString string
}

type thesis struct {
	URL, Title, Year string
}

func main() {
	mwURL := flag.String("mediawiki", "https://ims.ut.ee", "mediawiki URL")
	dspaceURL := flag.String("dspace", "https://dspace.ut.ee", "DSpace URL")
	lgName := flag.String("name", "", "login name of the bot for updating pages")
	lgPass := flag.String("pass", "", "login password of the bot for updating pages")
	tmplPath := flag.String("tmpl", "alumni-list.tmpl", "template for the ORCID list of publications")
	pageName := flag.String("page", "Alumni", "page name with alumni to parse and update")
	sectionTitle := flag.String("section", "PhD Students", "section title with alumni to parse and update")
	flag.Parse()
	if len(*mwURL) == 0 ||
		len(*lgName) == 0 ||
		len(*lgPass) == 0 ||
		len(*tmplPath) == 0 {
		log.Fatal("all flags are compulsory, use -h to see the documentation")
	}

	sectionIndex, err := getSectionIndex(*mwURL, *pageName, *sectionTitle)
	if err != nil {
		log.Fatal(err)
	}

	content, err := getPageSectionContent(*mwURL, *pageName, sectionIndex)
	if err != nil {
		log.Fatal(err)
	}

	users := findUsersFromTemplate(reTeamMember, content)

	// do not proceed if nobody to fetch and nothing to update
	if len(users) == 0 {
		return
	}

	advisors, err := getAdvisorsIMS(*mwURL)
	if err != nil {
		log.Fatal(err)
	}

	// download theses and upate users
	for _, v := range users {
		name := nameSplit(v.FullName)
		items, err := getUserThesesIMS(*dspaceURL, name[0], name[1], advisors)
		if err != nil {
			log.Fatal(err)
		}

		theses := make([]*thesis, len(items))
		for i, v := range items {
			theses[i] = &thesis{
				Title: v.Name,
				URL:   getItemMetaValue(v, "dc.identifier.uri").(string),
				Year:  getItemMetaValue(v, "dc.date.issued").(string), // TODO: parse and extract only a year
			}
		}
		v.Theses = theses
	}

	// append already existing alumni on the page
	existingAlumni := findUsersFromTemplate(reAlumnus, content)
	users = append(users, existingAlumni...)

	// render final markup with the Alumnus template, so after repeated
	// updates already updated users are not affected
	tmpl := template.Must(template.ParseFiles(*tmplPath))
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, users)
	if err != nil {
		log.Fatal(err)
	}

	// update page
	_, err = mw.UpdatePage(*mwURL, *pageName, buf.String(), "wikitext", *lgName, *lgPass, *sectionTitle)
	if err != nil {
		log.Fatal(err)
	}
}

// nameSplit splits arbitrarily complex names into two parts. Everything goes
// to the first name except the last part.
func nameSplit(s string) []string {
	parts := strings.Split(s, " ")
	// for compound names the rest goes to the first name
	firstName := strings.Join(parts[0:len(parts)-1], " ")
	lastName := parts[len(parts)-1]
	return []string{firstName, lastName}
}

func getSectionIndex(mwURL, page, section string) (string, error) {
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("format", "json")
	params.Set("page", page)
	params.Set("prop", "sections")

	resp, err := mw.Get(mwURL, params.Encode())
	if err != nil {
		return "", err
	}

	data, ok := resp["parse"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errResponseParsing, mwURL, page, section)
	}
	sectionsSlice, ok := data["sections"].([]interface{})
	if !ok {
		return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errResponseParsing, mwURL, page, section)
	}

	for _, v := range sectionsSlice {
		sectionMap, ok := v.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errResponseParsing, mwURL, page, section)
		}

		if title, ok := sectionMap["line"].(string); title == section && ok {
			return sectionMap["index"].(string), nil
		}
	}

	return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errNotFound, mwURL, page, section)
}

func getPageSectionContent(mwURL, page, sectionIndex string) (string, error) {
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("format", "json")
	params.Set("page", page)
	params.Set("prop", "wikitext")
	params.Set("section", sectionIndex)

	resp, err := mw.Get(mwURL, params.Encode())
	if err != nil {
		return "", err
	}

	data, ok := resp["parse"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errResponseParsing, mwURL, page, sectionIndex)
	}
	dataMap, ok := data["wikitext"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errResponseParsing, mwURL, page, sectionIndex)
	}
	markup, ok := dataMap["*"].(string)
	if !ok {
		return "", fmt.Errorf("%v, mwURL: %s, page: %s, section: %s", errResponseParsing, mwURL, page, sectionIndex)
	}

	return markup, nil
}

func findUsersFromTemplate(re *regexp.Regexp, s string) []*user {
	res := re.FindAllStringSubmatch(s, -1)

	users := make([]*user, len(res))
	for i, v := range res {
		// getting already existing data about an alumnus' theses
		var theses string
		if len(v) >= 5 {
			theses = strings.Trim(v[4], " ")
		}

		users[i] = &user{
			FullName: strings.Trim(v[2], " "),
			UserName: strings.Trim(v[1], " "),
			// removing the dot at the end if it use to append theses via the template
			Description:  strings.TrimRight(strings.Trim(v[3], " "), "."),
			ThesesString: theses,
		}
	}

	return users
}

func getUserTheses(apiURL, firstName, lastName string) ([]*item, error) {
	var name = fmt.Sprintf("%s, %s", lastName, firstName)

	mAuthor := meta{
		Key:   "dc.contributor.author",
		Value: name,
	}
	items, err := findItemsByMeta(apiURL, &mAuthor)
	if err != nil {
		return nil, err
	}

	mThesis := meta{
		Key:   "dc.type",
		Value: "Thesis",
	}
	items = filterItemsByMeta(items, &mThesis)

	return items, nil
}

func getUserThesesIMS(apiURL, firstName, lastName string, advisors []*user) ([]*item, error) {
	var name = fmt.Sprintf("%s, %s", lastName, firstName)

	mAuthor := &meta{
		Key:   "dc.contributor.author",
		Value: name,
	}
	items, err := findItemsByMeta(apiURL, mAuthor)
	if err != nil {
		return nil, err
	}

	// filter out only theses
	mThesis := &meta{
		Key:   "dc.type",
		Value: "Thesis",
	}
	items = filterItemsByMeta(items, mThesis)

	// filter out theses with advisors only from IMS
	var mAdvisor = &meta{Key: "dc.contributor.advisor"}
	var itemsWithAdvisors []*item
	for _, advisor := range advisors {
		nameParts := nameSplit(advisor.FullName)
		mAdvisor.Value = fmt.Sprintf("%s, %s", nameParts[1], nameParts[0])
		itemsWithAdvisors = filterItemsByMetaContains(items, mAdvisor)
		// we are insterested in at least one encounter with any advisor
		if len(itemsWithAdvisors) > 0 {
			break
		}
	}

	return itemsWithAdvisors, nil
}

func getAdvisorsIMS(mwURL string) ([]*user, error) {
	const (
		advisorsPage1    = "People"
		advisorsSection1 = "Staff"
		advisorsSection2 = "PhD Students"
		advisorsPage2    = "Alumni"
	)
	sectionIndexStaff, err := getSectionIndex(mwURL, advisorsPage1, advisorsSection1)
	if err != nil {
		return nil, err
	}
	sectionIndexPhD, err := getSectionIndex(mwURL, advisorsPage1, advisorsSection2)
	if err != nil {
		return nil, err
	}
	sectionIndexPhDAlumni, err := getSectionIndex(mwURL, advisorsPage2, advisorsSection2)
	if err != nil {
		return nil, err
	}

	contentStaff, err := getPageSectionContent(mwURL, advisorsPage1, sectionIndexStaff)
	if err != nil {
		return nil, err
	}
	contentPhD, err := getPageSectionContent(mwURL, advisorsPage1, sectionIndexPhD)
	if err != nil {
		return nil, err
	}
	contentPhDAlumni, err := getPageSectionContent(mwURL, advisorsPage2, sectionIndexPhDAlumni)
	if err != nil {
		return nil, err
	}

	advisorsStaff := findUsersFromTemplate(reTeamMember, contentStaff)
	advisorsPhD := findUsersFromTemplate(reTeamMember, contentPhD)
	advisorsPhDAlumni := findUsersFromTemplate(reAlumnus, contentPhDAlumni)

	advisors := []*user{}
	advisors = append(advisors, advisorsStaff...)
	advisors = append(advisors, advisorsPhD...)
	advisors = append(advisors, advisorsPhDAlumni...)
	return advisors, nil
}
