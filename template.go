package menu

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/cybozu-go/netutil"
	"golang.org/x/crypto/bcrypt"
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
	nodeNetworks         []*net.IPNet
}

// Node is template args for Node
type Node struct {
	Name             string
	Addresses        []string
	SystemdAddresses []string
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
			Host string
			VM   string
		}
	}
	Racks   []Rack
	Spines  []Spine
	CS      VMResource
	SS      VMResource
	Boot    VMResource
	Account struct {
		Name         string
		PasswordHash string
	}
}

// VMResource is args to specify vm resource
type VMResource struct {
	CPU    int
	Memory string
}

// ToTemplateArgs is converter Menu to TemplateArgs
func ToTemplateArgs(menu *Menu) (*TemplateArgs, error) {
	var templateArgs TemplateArgs
	templateArgs.Account.Name = menu.Account.UserName
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(menu.Account.Password), 10)
	if err != nil {
		return nil, err
	}
	templateArgs.Account.PasswordHash = string(passwordHash)
	templateArgs.Network.External.Host = addToIPNet(menu.Network.External, 1)
	templateArgs.Network.External.VM = addToIPNet(menu.Network.External, 2)

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
			return nil, errors.New("invalid node type")
		}
	}

	templateArgs.Racks = make([]Rack, len(menu.Inventory.Rack))
	for rackIdx, rackMenu := range menu.Inventory.Rack {
		rack := &templateArgs.Racks[rackIdx]
		rack.Name = fmt.Sprintf("rack%d", rackIdx)
		rack.nodeNetworks = make([]*net.IPNet, 3)
		for i := 0; i < 3; i++ {
			rack.nodeNetworks[i] = makeNodeNetwork(menu.Network.Node, rackIdx*3+i)
		}

		constructToRAddresses(rack, rackIdx, menu)
		constructBootAddresses(rack, rackIdx, menu)

		for csIdx := 0; csIdx < rackMenu.CS; csIdx++ {
			node := constructNode("cs", csIdx, 3, rack)
			rack.CSList = append(rack.CSList, node)
		}
		for ssIdx := 0; ssIdx < rackMenu.SS; ssIdx++ {
			node := constructNode("ss", ssIdx, 3+rackMenu.CS, rack)
			rack.SSList = append(rack.SSList, node)
		}
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spine := &templateArgs.Spines[spineIdx]
		spine.Name = fmt.Sprintf("spine%d", spineIdx+1)

		// {external network} + {tor per rack} * {rack}
		numRack := len(menu.Inventory.Rack)
		spine.Addresses = make([]string, 1+torPerRack*numRack)
		spine.Addresses[0] = addToIPNet(menu.Network.External, 3+spineIdx)
		for rackIdx := range menu.Inventory.Rack {
			spine.Addresses[rackIdx*torPerRack+1] = addToIP(
				menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2, 31)
			spine.Addresses[rackIdx*torPerRack+2] = addToIP(
				menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2+2, 31)
		}
	}

	return &templateArgs, nil
}

func constructNode(basename string, idx int, offsetStart int, rack *Rack) Node {
	node := Node{}
	node.Name = fmt.Sprintf("%v%d", basename, idx+1)
	node.Addresses = make([]string, 3)
	node.SystemdAddresses = make([]string, 3)
	offset := offsetStart + idx + 1

	node.Addresses[0] = addToIP(rack.nodeNetworks[0].IP, offset, 32)
	node.Addresses[1] = addToIPNet(rack.nodeNetworks[1], offset)
	node.Addresses[2] = addToIPNet(rack.nodeNetworks[2], offset)
	node.SystemdAddresses[0] = removePrefixSize(node.Addresses[0])
	node.SystemdAddresses[1] = rack.BootSystemdAddresses[1]
	node.SystemdAddresses[2] = rack.BootSystemdAddresses[2]
	return node
}

func constructBootAddresses(rack *Rack, rackIdx int, menu *Menu) {
	rack.BootAddresses = make([]string, 4)
	rack.BootAddresses[0] = addToIP(rack.nodeNetworks[0].IP, 3, 32)
	rack.BootAddresses[1] = addToIPNet(rack.nodeNetworks[1], 3)
	rack.BootAddresses[2] = addToIPNet(rack.nodeNetworks[2], 3)
	rack.BootAddresses[3] = addToIP(menu.Network.Bastion.IP, rackIdx, 32)

	rack.BootSystemdAddresses = make([]string, 3)
	rack.BootSystemdAddresses[0] = removePrefixSize(rack.BootAddresses[0])
	rack.BootSystemdAddresses[1] = removePrefixSize(addToIPNet(rack.nodeNetworks[1], 1))
	rack.BootSystemdAddresses[2] = removePrefixSize(addToIPNet(rack.nodeNetworks[2], 1))
}

func constructToRAddresses(rack *Rack, rackIdx int, menu *Menu) {
	numRack := len(menu.Inventory.Rack)
	rack.ToR1Addresses = make([]string, menu.Inventory.Spine+1)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR1Addresses[spineIdx] = addToIP(
			menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2+1, 31)
	}
	rack.ToR1Addresses[menu.Inventory.Spine] = addToIPNet(
		rack.nodeNetworks[1], 1)

	rack.ToR2Addresses = make([]string, menu.Inventory.Spine+1)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR2Addresses[spineIdx] = addToIP(
			menu.Network.SpineTor, (spineIdx*numRack+rackIdx)*torPerRack*2+3, 31)
	}
	rack.ToR2Addresses[menu.Inventory.Spine] = addToIPNet(
		rack.nodeNetworks[2], 1)
}

func removePrefixSize(input string) string {
	return strings.Split(input, "/")[0]
}

func addToIPNet(netAddr *net.IPNet, offset int) string {
	ipint := netutil.IP4ToInt(netAddr.IP) + uint32(offset)
	prefixSize, _ := netAddr.Mask.Size()
	return fmt.Sprintf("%s/%d", netutil.IntToIP4(ipint).String(), prefixSize)
}

func addToIP(netIP net.IP, offset int, prefixSize int) string {
	ipint := netutil.IP4ToInt(netIP) + uint32(offset)
	return fmt.Sprintf("%s/%d", netutil.IntToIP4(ipint).String(), prefixSize)
}

func makeNodeNetwork(base *net.IPNet, nodeIdx int) *net.IPNet {
	prefixSize, _ := base.Mask.Size()
	offset := 1 << uint(32-prefixSize) * nodeIdx
	ipint := netutil.IP4ToInt(base.IP) + uint32(offset)
	return &net.IPNet{netutil.IntToIP4(ipint), base.Mask}
}
