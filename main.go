package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"regexp"
	"strings"

	mw "bitbucket.org/iharsuvorau/mediawiki"
)

var (
	// errResponseParsing signifies the type conversion error. Usually, it
	// means there is unexpected content in the response.
	errResponseParsing = errors.New("unexpected response")

	// errNotFound signifies not found content.
	errNotFound = errors.New("not found")
)

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

	users := findUsersFromTemplate(content)

	// do not proceed if nobody to fetch and nothing to update
	if len(users) == 0 {
		return
	}

	// download theses and upate users
	for _, v := range users {
		name := nameSplit(v.FullName)
		items, err := getUserTheses(*dspaceURL, name[0], name[1])
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

type user struct {
	FullName, UserName, Description string
	Theses                          []*thesis
}
type thesis struct {
	URL, Title, Year string
}

// findUsersFromTemplate returns users with the TeamMember template from a page
// with a bit modified content for further usage in the Alumnus template
// (removing of whitespaces from both sides, removing of punctuation at the end
// of description line).
func findUsersFromTemplate(s string) []*user {
	reg := regexp.MustCompile(`{{\s*TeamMember\s*\|\s*(.*)\s*\|\s*(.*)\s*\|\s*(.*)}}\s*`) // username, name, description
	res := reg.FindAllStringSubmatch(s, -1)

	users := make([]*user, len(res))
	for i, v := range res {
		users[i] = &user{
			FullName: strings.Trim(v[2], " "),
			UserName: strings.Trim(v[1], " "),
			// removing the dot at the end if it use to append theses via the template
			Description: strings.TrimRight(strings.Trim(v[3], " "), "."),
		}
	}

	return users
}
