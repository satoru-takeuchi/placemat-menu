package menu

import (
	"errors"
	"fmt"
	"net"

	"github.com/cybozu-go/netutil"
)

// Rack is template args for rack
type Rack struct {
	Name   string
	CSList []Node
	SSList []Node
}

// Node is template args for Node
type Node struct {
	Name string
}

// Spine is template args for Spine
type Spine struct {
	Name      string
	Addresses []string
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

// VMResource is args to specify vm resource
type VMResource struct {
	CPU    int
	Memory string
}

// ToTemplateArgs is converter Menu to TemplateArgs
func ToTemplateArgs(menu *Menu) (*TemplateArgs, error) {
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
			return nil, errors.New("Invalid node type")
		}
	}

	templateArgs.Racks = make([]Rack, len(menu.Inventory.Rack))
	for index, rackMenu := range menu.Inventory.Rack {
		templateArgs.Racks[index].Name = fmt.Sprintf("rack%d", index)
		for csidx := 0; csidx < rackMenu.CS; csidx++ {
			templateArgs.Racks[index].CSList = append(templateArgs.Racks[index].CSList,
				Node{fmt.Sprintf("cs%d", csidx+1)})
		}
		for ssidx := 0; ssidx < rackMenu.SS; ssidx++ {
			templateArgs.Racks[index].SSList = append(templateArgs.Racks[index].SSList,
				Node{fmt.Sprintf("ss%d", ssidx+1)})
		}
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spine := &templateArgs.Spines[spineIdx]
		spine.Name = fmt.Sprintf("spine%d", spineIdx+1)

		// {external network} + {tor per rack} * {rack}
		numRack := len(menu.Inventory.Rack)
		torPerRack := 2

		spine.Addresses = make([]string, 1+torPerRack*numRack)

		spine.Addresses[0] = makeHostAddressFromNetworkAddress(menu.Network.External, 3+spineIdx)
		for rackIdx := range menu.Inventory.Rack {
			spine.Addresses[rackIdx*torPerRack+1] = makeHostAddressFromIPAddress(
				&menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2, 31)
			spine.Addresses[rackIdx*torPerRack+2] = makeHostAddressFromIPAddress(
				&menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2+2, 31)
		}
	}

	return &templateArgs, nil
}

func makeHostAddressFromNetworkAddress(netAddr *net.IPNet, offset int) string {
	ipint := netutil.IP4ToInt(netAddr.IP) + uint32(offset)
	prefixSize, _ := netAddr.Mask.Size()
	return fmt.Sprintf("%s/%d", netutil.IntToIP4(ipint).String(), prefixSize)
}
func makeHostAddressFromIPAddress(netIP *net.IP, offset int, prefixSize int) string {
	ipint := netutil.IP4ToInt(*netIP) + uint32(offset)
	return fmt.Sprintf("%s/%d", netutil.IntToIP4(ipint).String(), prefixSize)
}
