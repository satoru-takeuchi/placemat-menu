package menu

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/cybozu-go/netutil"
)

const (
	torPerRack = 2
)

// Rack is template args for rack
type Rack struct {
	Name                 string
	BootAddresses        []string
	BootSystemdAddresses []string
	ToR1Addresses        []string
	ToR2Addresses        []string
	CSList               []Node
	SSList               []Node
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
	for rackIdx, rackMenu := range menu.Inventory.Rack {
		rack := &templateArgs.Racks[rackIdx]
		rack.Name = fmt.Sprintf("rack%d", rackIdx)

		constructToRAddresses(rack, rackIdx, menu)
		constructBootAddresses(rack, rackIdx, menu)

		for csIdx := 0; csIdx < rackMenu.CS; csIdx++ {
			rack.CSList = append(rack.CSList, Node{fmt.Sprintf("cs%d", csIdx+1)})
		}
		for ssIdx := 0; ssIdx < rackMenu.SS; ssIdx++ {
			rack.SSList = append(rack.SSList, Node{fmt.Sprintf("ss%d", ssIdx+1)})
		}
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spine := &templateArgs.Spines[spineIdx]
		spine.Name = fmt.Sprintf("spine%d", spineIdx+1)

		// {external network} + {tor per rack} * {rack}
		numRack := len(menu.Inventory.Rack)
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

func constructBootAddresses(rack *Rack, rackIdx int, menu *Menu) {
	node0Network := makeNodeNetwork(menu.Network.Node, rackIdx*3)
	node1Network := makeNodeNetwork(menu.Network.Node, rackIdx*3+1)
	node2Network := makeNodeNetwork(menu.Network.Node, rackIdx*3+2)

	rack.BootAddresses = make([]string, 4)
	rack.BootAddresses[0] = makeHostAddressFromIPAddress(&node0Network.IP, 3, 32)
	rack.BootAddresses[1] = makeHostAddressFromNetworkAddress(node1Network, 3)
	rack.BootAddresses[2] = makeHostAddressFromNetworkAddress(node2Network, 3)
	rack.BootAddresses[3] = makeHostAddressFromIPAddress(&menu.Network.Bastion.IP, rackIdx, 32)

	rack.BootSystemdAddresses = make([]string, 3)
	rack.BootSystemdAddresses[0] = removePrefixSize(rack.BootAddresses[0])
	rack.BootSystemdAddresses[1] = removePrefixSize(makeHostAddressFromNetworkAddress(node1Network, 1))
	rack.BootSystemdAddresses[2] = removePrefixSize(makeHostAddressFromNetworkAddress(node2Network, 1))
}

func constructToRAddresses(rack *Rack, rackIdx int, menu *Menu) {
	numRack := len(menu.Inventory.Rack)
	rack.ToR1Addresses = make([]string, menu.Inventory.Spine+1)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR1Addresses[spineIdx] = makeHostAddressFromIPAddress(
			&menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2+1, 31)
	}
	node1Network := makeNodeNetwork(menu.Network.Node, rackIdx*3+1)
	rack.ToR1Addresses[menu.Inventory.Spine] = makeHostAddressFromNetworkAddress(
		node1Network, 1)

	rack.ToR2Addresses = make([]string, menu.Inventory.Spine+1)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR2Addresses[spineIdx] = makeHostAddressFromIPAddress(
			&menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2+3, 31)
	}
	node2Network := makeNodeNetwork(menu.Network.Node, rackIdx*3+2)
	rack.ToR2Addresses[menu.Inventory.Spine] = makeHostAddressFromNetworkAddress(
		node2Network, 1)
}

func removePrefixSize(input string) string {
	return strings.Split(input, "/")[0]
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

func makeNodeNetwork(base *net.IPNet, nodeIdx int) *net.IPNet {
	prefixSize, _ := base.Mask.Size()
	offset := 1 << uint(32-prefixSize) * nodeIdx
	ipint := netutil.IP4ToInt(base.IP) + uint32(offset)
	return &net.IPNet{netutil.IntToIP4(ipint), base.Mask}
}
