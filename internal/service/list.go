package service

import (
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/utils"
	"fmt"
	"sort"
	"strings"
)

// ListNodes lists all available nodes
func (s *SpeedTest) ListNodes() error {
	if len(s.nodes) == 0 {
		return fmt.Errorf("node list is empty")
	}

	headers := []string{"名称", "运营商", "节点类型", "节点ID"}
	table := utils.NewTable(headers)

	table.EnableAutoMerge()
	table.SortBy([]string{"节点类型", "运营商"})

	for id, node := range s.nodes {
		table.AddRow([]string{
			node.Name.Zh,
			node.Isp.Zh,
			strings.ToUpper(node.GeoInfo.Type),
			id,
		})
	}

	if len(s.nodes) > 25 {
		table.SetPageSize(25)
	}

	table.Print()
	return nil
}

// getAvailableIDs gets all available node IDs
func getAvailableIDs(nodes []models.Node) []string {
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.Id)
	}
	return ids
}

// GetNodeIDByInput gets node ID by either numeric or string input
func (s *SpeedTest) GetNodeIDByInput(input string) (string, error) {
	// Try to parse as a number
	var numID int
	if _, err := fmt.Sscanf(input, "%d", &numID); err == nil {
		// If it's a number, iterate through sorted nodes to find the corresponding one
		index := 1
		// Sort nodes by type and ISP to match table display
		sortedNodes := s.getSortedNodes()
		for _, node := range sortedNodes {
			if index == numID {
				return node.Id, nil
			}
			index++
		}
		return "", fmt.Errorf("无效的序号: %d", numID)
	}

	// If it's not a number, check if it's a valid node ID
	for id := range s.nodes {
		if id == input {
			return input, nil
		}
	}
	return "", fmt.Errorf("无效的节点ID: %s", input)
}

// getSortedNodes returns nodes sorted by type and ISP to match table display
func (s *SpeedTest) getSortedNodes() []models.Node {
	nodes := make([]models.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	// Sort by type and ISP to match table display
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].GeoInfo.Type != nodes[j].GeoInfo.Type {
			return nodes[i].GeoInfo.Type < nodes[j].GeoInfo.Type
		}
		return nodes[i].Isp.Zh < nodes[j].Isp.Zh
	})

	return nodes
}
