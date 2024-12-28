package service

import (
	"aqua-speed-tools/internal/models"
	"aqua-speed-tools/internal/utils"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
)

// ListNodes lists all available nodes
func (s *SpeedTest) ListNodes() error {
	if len(s.nodes) == 0 {
		return fmt.Errorf("node list is empty")
	}

	headers := []string{"名称", "运营商", "节点类型", "测速所需流量", "节点ID"}
	table := utils.NewTable(headers)

	table.EnableAutoMerge()
	table.SortBy([]string{"节点类型", "运营商"})

	for id, node := range s.nodes {
		size := humanize.Bytes(uint64(node.Size.Value) * 1000 * 1000 * 4)
		table.AddRow([]string{
			node.Name.Zh,
			node.Isp.Zh,
			strings.ToUpper(node.GeoInfo.Type),
			size,
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
