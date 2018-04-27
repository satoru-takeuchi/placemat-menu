package menu

import (
	"fmt"

	"github.com/cybozu-go/netutil"
)

// Rack is template args for rack
type Rack struct {
	Name string
	CSs  []Node
	SSs  []Node
}

type Node struct {
	Name string
}

// Spine is template args for Spine
type Spine struct {
	Name string
}

// TemplateArgs is args for cluster.yml
type TemplateArgs struct {
	Network struct {
		External struct {
			HostVM string
		}
	}
	Racks  []Rack
	Spines []Spine
	CS     VMResource
	SS     VMResource
	Boot   VMResource
}
type VMResource struct {
	CPU    int
	Memory string
}

// MenuToTemplateArgs is converter Menu to TemplateArgs
func MenuToTemplateArgs(menu *Menu) (TemplateArgs, error) {
	var templateArgs TemplateArgs

	extnet := netutil.IP4ToInt(menu.Network.External.IP)
	hostvmip := extnet + 1
	hostvmprefix, _ := menu.Network.External.Mask.Size()
	hostvmipnet := fmt.Sprintf("%s/%d", netutil.IntToIP4(hostvmip).String(), hostvmprefix)
	templateArgs.Network.External.HostVM = hostvmipnet

	for _, node := range menu.Nodes {
		switch node.Type {
		case CSNode:
			templateArgs.CS.Memory = node.Memory
			templateArgs.CS.CPU = node.CPU
		case SSNode:
			templateArgs.SS.Memory = node.Memory
			templateArgs.SS.CPU = node.CPU
		case BootNode:
			templateArgs.Boot.Memory = node.Memory
			templateArgs.Boot.CPU = node.CPU
		default:
			continue
		}
	}

	templateArgs.Racks = make([]Rack, len(menu.Inventory.Rack))
	for index, rackMenu := range menu.Inventory.Rack {
		templateArgs.Racks[index].Name = fmt.Sprintf("rack%d", index)
		for csidx := 0; csidx < rackMenu.CS; csidx++ {
			templateArgs.Racks[index].CSs = append(templateArgs.Racks[index].CSs,
				Node{fmt.Sprintf("cs%d", csidx)})
		}
		for ssidx := 0; ssidx < rackMenu.CS; ssidx++ {
			templateArgs.Racks[index].SSs = append(templateArgs.Racks[index].SSs,
				Node{fmt.Sprintf("ss%d", ssidx)})
		}
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for index := 0; index < menu.Inventory.Spine; index++ {
		templateArgs.Spines[index].Name = fmt.Sprintf("spine%d", index+1)
	}

	return templateArgs, nil
}
