package models

import (
	"encoding/json"
	"fmt"
)

// Size in MB, with custom parsing
type Size struct {
	Value int64 `json:"value"`
}

// UnmarshalJSON implements custom parsing for Size
func (s *Size) UnmarshalJSON(data []byte) error {
	// First try to parse directly as a number
	var value int64
	if err := json.Unmarshal(data, &value); err == nil {
		s.Value = value
		return nil
	}

	// If failed, try to parse as an object
	var objMap map[string]json.Number
	if err := json.Unmarshal(data, &objMap); err != nil {
		return err
	}

	if val, ok := objMap["value"]; ok {
		value, err := val.Int64()
		if err != nil {
			return fmt.Errorf("Size.Value is invalid: %v", err)
		}
		s.Value = value
		return nil
	}

	return fmt.Errorf("size.value field not found")
}

type Node struct {
	Id   string `json:"id"`
	Name struct {
		Zh string `json:"zh"`
		En string `json:"en"`
	} `json:"name"`
	Size Size `json:"size"`
	Isp  struct {
		Zh string `json:"zh"`
		En string `json:"en"`
	} `json:"isp"`
	Url     string `json:"url"`
	Threads int    `json:"threads"`
	Type    string `json:"type"`
	GeoInfo struct {
		CountryCode string `json:"countryCode"`
		Region      string `json:"region"`
		City        string `json:"city"`
		Type        string `json:"type"`
	} `json:"geoInfo"`
}

type NodeList map[string]Node
