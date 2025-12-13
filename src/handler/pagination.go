package handler

import (
	"encoding/json"
	"net/http"
)

// ParseRangeParams extracts pagination offsets from a `range` query param following the
// react-admin convention: [start, end]. If parsing fails, it defaults to 0 offset and
// limit 10.
func ParseRangeParams(r *http.Request) (offset, limit int) {
	rangeStr := r.URL.Query().Get("range")
	var rangeVals [2]int
	if rangeStr == "" || json.Unmarshal([]byte(rangeStr), &rangeVals) != nil {
		return 0, 10 // default
	}
	return rangeVals[0], rangeVals[1] - rangeVals[0] + 1
}

// ParseSortParams reads the `sort` query param formatted as [field, direction] and
// provides a default of ("id", "ASC") when absent or malformed.
func ParseSortParams(r *http.Request) (field, direction string) {
	sortStr := r.URL.Query().Get("sort")
	var sortVals [2]string
	if sortStr == "" || json.Unmarshal([]byte(sortStr), &sortVals) != nil {
		return "id", "ASC"
	}
	return sortVals[0], sortVals[1]
}

// ParseFilterParams unmarshals the `filter` query param into a string map. When the
// filter cannot be parsed, it returns nil.
func ParseFilterParams(r *http.Request) map[string]string {
	filterStr := r.URL.Query().Get("filter")
	if filterStr == "" {
		return nil
	}
	var filters map[string]string
	if err := json.Unmarshal([]byte(filterStr), &filters); err != nil {
		return nil
	}
	return filters
}
