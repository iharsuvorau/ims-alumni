package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type meta struct {
	Key       string `json:"key,omitempty"`
	Value     string `json:"value,omitempty"`
	Element   string `json:"element,omitempty"`
	Qualifier string `json:"qualifier,omitempty"`
	Schema    string `json:"schema,omitempty"`
}

type item struct {
	UUID                 string      `json:"uuid,omitempty"`
	Name                 string      `json:"name,omitempty"`
	Handle               string      `json:"handle,omitempty"`
	Type                 string      `json:"type,omitempty"`
	Expand               []string    `json:"expand,omitempty"`
	LastModified         string      `json:"lastModified,omitempty"`
	ParentCollection     interface{} `json:"parentCollection,omitempty"`
	ParentCollectionList interface{} `json:"parentCollectionList,omitempty"`
	ParentCommunityList  interface{} `json:"parentCommunityList,omitempty"`
	Bitstreams           interface{} `json:"bitstreams,omitempty"`
	Withdrawn            interface{} `json:"withdrawn,omitempty"`
	Archived             interface{} `json:"archived,omitempty"`
	Link                 string      `json:"link,omitempty"`
	Metadata             []*meta     `json:"metadata,omitempty"`
}

func findItemsByMeta(apiUrl string, q *query) ([]*item, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(q)
	if err != nil {
		return nil, err
	}

	const contentType = "application/json"
	resp, err := http.Post(apiUrl, contentType, &buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	items := []*item{}
	err = json.NewDecoder(resp.Body).Decode(&items)

	return items, err
}

func main() {
	const apiUrl = "https://dspace.ut.ee/rest/items/find-by-metadata-field?expand=metadata"

	qAuthor := query{
		Key: "dc.contributor.author",
		//Value: "Aabloo, Alvo",
		Value: "Must, Indrek",
	}

	items, err := findItemsByMeta(apiUrl, &qAuthor)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range items {
		var (
			isThesis bool
			idUrl    string
		)
		for _, m := range v.Metadata {
			if m.Key == "dc.identifier.uri" {
				idUrl = m.Value
			}
			if m.Key == "dc.type" && strings.ToLower(m.Value) == "thesis" {
				isThesis = true
			}
		}

		if isThesis {
			fmt.Printf("%s @ %s", v.Name, idUrl)
		}
	}
}
