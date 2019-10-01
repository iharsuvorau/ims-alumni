package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func findItemsByMeta(dspaceBaseURL string, m *meta) ([]*item, error) {
	const contentType = "application/json"
	var apiURL = fmt.Sprintf("%s/rest/items/find-by-metadata-field?expand=metadata",
		strings.TrimRight(dspaceBaseURL, "/"))
	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(m)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(apiURL, contentType, &buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	items := []*item{}
	err = json.NewDecoder(resp.Body).Decode(&items)

	return items, err
}

// filterItemsByMeta compares items meta fields with the provided meta by
// strings.EqualFold functin for exact match.
func filterItemsByMeta(items []*item, m *meta) []*item {
	results := []*item{}

	for _, v := range items {
		var isMatch bool
		for _, vv := range v.Metadata {
			if vv.Key == m.Key && strings.EqualFold(vv.Value, m.Value) {
				isMatch = true
				break
			}
		}

		if isMatch {
			results = append(results, v)
		}
	}

	return results
}

// filterItemsByMetaContains is the same as filterItemsByMeta but compares meta
// values not with the strings.EqualFold function, but with the
// strings.Contains function for not exact match.
func filterItemsByMetaContains(items []*item, m *meta) []*item {
	results := []*item{}

	for _, v := range items {
		var isMatch bool
		for _, vv := range v.Metadata {
			if vv.Key == m.Key && strings.Contains(vv.Value, m.Value) {
				isMatch = true
				break
			}
		}

		if isMatch {
			results = append(results, v)
		}
	}

	return results
}

// getItemMetaValue returns the first found key value in
// item.Metadata. So not very useful to extract authors, for example,
// becauses there can be several of them, but for single values that's
// enough.
func getItemMetaValue(item *item, key string) interface{} {
	for _, m := range item.Metadata {
		if m.Key == key {
			return m.Value
		}
	}
	return nil
}
