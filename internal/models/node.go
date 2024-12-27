package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Size in MB, with custom parsing
type Size struct {
	Value int64 `json:"value"`
}

// UnmarshalJSON implements custom parsing for Size
func (s *Size) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data for Size unmarshal")
	}

	// First try to parse directly as a number
	var value int64
	if err := json.Unmarshal(data, &value); err == nil {
		if value < 0 {
			return fmt.Errorf("size value cannot be negative: %d", value)
		}
		s.Value = value
		return nil
	}

	// If failed, try to parse as an object
	var objMap map[string]json.Number
	if err := json.Unmarshal(data, &objMap); err != nil {
		return fmt.Errorf("failed to parse Size: %v", err)
	}

	if val, ok := objMap["value"]; ok {
		value, err := val.Int64()
		if err != nil {
			return fmt.Errorf("invalid Size.Value: %v", err)
		}
		if value < 0 {
			return fmt.Errorf("size value cannot be negative: %d", value)
		}
		s.Value = value
		return nil
	}

	return fmt.Errorf("size.value field not found")
}

type GeoInfo struct {
	CountryCode string  `json:"countryCode"`
	Region      *string `json:"region"`
	City        *string `json:"city"`
	Type        string  `json:"type"`
}

// Validate checks if GeoInfo fields are valid
func (g *GeoInfo) Validate() error {
	if g.CountryCode == "" {
		return fmt.Errorf("countryCode cannot be empty")
	}
	if len(g.CountryCode) != 2 {
		return fmt.Errorf("countryCode must be 2 characters: %s", g.CountryCode)
	}
	if g.Type == "" {
		return fmt.Errorf("type cannot be empty")
	}
	return nil
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
	Url     string  `json:"url"`
	Threads uint8   `json:"threads"`
	Type    string  `json:"type"`
	GeoInfo GeoInfo `json:"geoInfo"`
}

// Validate checks if Node fields are valid
func (n *Node) Validate() error {
	if n.Id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	if n.Name.Zh == "" && n.Name.En == "" {
		return fmt.Errorf("at least one name (zh or en) must be provided")
	}

	if n.Size.Value <= 0 {
		return fmt.Errorf("size must be positive")
	}

	if n.Isp.Zh == "" && n.Isp.En == "" {
		return fmt.Errorf("at least one ISP name (zh or en) must be provided")
	}

	if n.Url != "" && !strings.HasPrefix(n.Url, "http") {
		return fmt.Errorf("invalid URL format: %s", n.Url)
	}

	if n.Threads == 0 {
		return fmt.Errorf("threads cannot be zero")
	}

	if n.Type == "" {
		return fmt.Errorf("type cannot be empty")
	}

	if err := n.GeoInfo.Validate(); err != nil {
		return fmt.Errorf("invalid geoInfo: %v", err)
	}

	return nil
}

type NodeList map[string]Node

// Validate checks if all nodes in the NodeList are valid
func (nl NodeList) Validate() error {
	if len(nl) == 0 {
		return fmt.Errorf("nodeList cannot be empty")
	}

	for id, node := range nl {
		if id != node.Id {
			return fmt.Errorf("node id mismatch: map key %s != node id %s", id, node.Id)
		}
		if err := node.Validate(); err != nil {
			return fmt.Errorf("invalid node %s: %v", id, err)
		}
	}

	return nil
}
