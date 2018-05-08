package menu

import (
	"errors"
	"fmt"
	"net"

	"github.com/cybozu-go/netutil"
	"golang.org/x/crypto/bcrypt"
)

const (
	torPerRack = 2

	offsetExtnetHost   = 1
	offsetExtnetVM     = 2
	offsetExtnetSpines = 3

	offsetNodenetToR     = 1
	offsetNodenetBoot    = 3
	offsetNodenetServers = 4

	offsetASNExtVM = -2
	offsetASNSpine = -1
)

// Rack is template args for rack
type Rack struct {
	Name                  string
	ASN                   int
	NodeNetworkPrefixSize int
	BootAddresses         []*net.IPNet
	BootSystemdAddresses  []*net.IPNet
	ToR1SpineAddresses    []*net.IPNet
	ToR1NodeAddress       *net.IPNet
	ToR1NodeInterface     string
	ToR2SpineAddresses    []*net.IPNet
	ToR2NodeAddress       *net.IPNet
	ToR2NodeInterface     string
	CSList                []Node
	SSList                []Node
	nodeNetworks          []*net.IPNet
}

// Node is template args for Node
type Node struct {
	Name             string
	Addresses        []*net.IPNet
	SystemdAddresses []*net.IPNet
}

// Spine is template args for Spine
type Spine struct {
	Name          string
	ExtnetAddress *net.IPNet
	ToRAddresses  []*net.IPNet
}

// ToR1Address returns spine's IP address connected from ToR-1 in the specified rack
func (s Spine) ToR1Address(rackIdx int) *net.IPNet {
	return s.ToRAddresses[rackIdx*2]
}

// ToR2Address returns spine's IP address connected from ToR-2 in the specified rack
func (s Spine) ToR2Address(rackIdx int) *net.IPNet {
	return s.ToRAddresses[rackIdx*2+1]
}

// TemplateArgs is args for cluster.yml
type TemplateArgs struct {
	Network struct {
		External struct {
			Host *net.IPNet
			VM   *net.IPNet
		}
		Exposed struct {
			Bastion      *net.IPNet
			LoadBalancer *net.IPNet
			Ingress      *net.IPNet
		}
		ASNExtVM int
		ASNSpine int
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

// BIRDRackTemplateArgs is args to generate bird config for each rack
type BIRDRackTemplateArgs struct {
	Args    TemplateArgs
	RackIdx int
}

// BIRDSpineTemplateArgs is args to generate bird config for each spine
type BIRDSpineTemplateArgs struct {
	Args     TemplateArgs
	SpineIdx int
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

	setNetworkArgs(&templateArgs, menu)

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

	numRack := len(menu.Inventory.Rack)

	spineToRackBases := make([][]net.IP, menu.Inventory.Spine)
	spineTorInt := netutil.IP4ToInt(menu.Network.SpineTor)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spineToRackBases[spineIdx] = make([]net.IP, numRack)
		for rackIdx := range menu.Inventory.Rack {
			offset := uint32((spineIdx*numRack + rackIdx) * torPerRack * 2)
			spineToRackBases[spineIdx][rackIdx] = netutil.IntToIP4(spineTorInt + offset)
		}
	}

	templateArgs.Racks = make([]Rack, numRack)
	for rackIdx, rackMenu := range menu.Inventory.Rack {
		rack := &templateArgs.Racks[rackIdx]
		rack.Name = fmt.Sprintf("rack%d", rackIdx)
		rack.ASN = menu.Network.ASNBase + rackIdx
		rack.nodeNetworks = make([]*net.IPNet, 3)
		for i := 0; i < 3; i++ {
			rack.nodeNetworks[i] = makeNodeNetwork(menu.Network.Node, rackIdx*3+i)
		}

		constructToRAddresses(rack, rackIdx, menu, spineToRackBases)
		constructBootAddresses(rack, rackIdx, menu)
		prefixSize, _ := menu.Network.Node.Mask.Size()
		rack.NodeNetworkPrefixSize = prefixSize

		for csIdx := 0; csIdx < rackMenu.CS; csIdx++ {
			node := buildNode("cs", csIdx, offsetNodenetServers, rack)
			rack.CSList = append(rack.CSList, node)
		}
		for ssIdx := 0; ssIdx < rackMenu.SS; ssIdx++ {
			node := buildNode("ss", ssIdx, offsetNodenetServers+rackMenu.CS, rack)
			rack.SSList = append(rack.SSList, node)
		}
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spine := &templateArgs.Spines[spineIdx]
		spine.Name = fmt.Sprintf("spine%d", spineIdx+1)

		// {external network} + {tor per rack} * {rack}
		spine.ExtnetAddress = addToIPNet(menu.Network.External, offsetExtnetSpines+spineIdx)
		spine.ToRAddresses = make([]*net.IPNet, torPerRack*numRack)
		for rackIdx := range menu.Inventory.Rack {
			spine.ToRAddresses[rackIdx*torPerRack] = addToIP(spineToRackBases[spineIdx][rackIdx], 0, 31)
			spine.ToRAddresses[rackIdx*torPerRack+1] = addToIP(spineToRackBases[spineIdx][rackIdx], 2, 31)
		}
	}

	return &templateArgs, nil
}

func setNetworkArgs(templateArgs *TemplateArgs, menu *Menu) {
	templateArgs.Network.External.Host = addToIPNet(menu.Network.External, offsetExtnetHost)
	templateArgs.Network.External.VM = addToIPNet(menu.Network.External, offsetExtnetVM)
	templateArgs.Network.ASNExtVM = menu.Network.ASNBase + offsetASNExtVM
	templateArgs.Network.ASNSpine = menu.Network.ASNBase + offsetASNSpine
	templateArgs.Network.Exposed.Bastion = menu.Network.Bastion
	templateArgs.Network.Exposed.LoadBalancer = menu.Network.LoadBalancer
	templateArgs.Network.Exposed.Ingress = menu.Network.Ingress
}

func buildNode(basename string, idx int, offsetStart int, rack *Rack) Node {
	node := Node{}
	node.Name = fmt.Sprintf("%v%d", basename, idx+1)
	node.Addresses = make([]*net.IPNet, 3)
	node.SystemdAddresses = make([]*net.IPNet, 3)
	offset := offsetStart + idx

	node.Addresses[0] = addToIP(rack.nodeNetworks[0].IP, offset, 32)
	node.Addresses[1] = addToIPNet(rack.nodeNetworks[1], offset)
	node.Addresses[2] = addToIPNet(rack.nodeNetworks[2], offset)
	node.SystemdAddresses[0] = node.Addresses[0]
	node.SystemdAddresses[1] = rack.BootSystemdAddresses[1]
	node.SystemdAddresses[2] = rack.BootSystemdAddresses[2]
	return node
}

func constructBootAddresses(rack *Rack, rackIdx int, menu *Menu) {
	rack.BootAddresses = make([]*net.IPNet, 4)
	rack.BootAddresses[0] = addToIP(rack.nodeNetworks[0].IP, offsetNodenetBoot, 32)
	rack.BootAddresses[1] = addToIPNet(rack.nodeNetworks[1], offsetNodenetBoot)
	rack.BootAddresses[2] = addToIPNet(rack.nodeNetworks[2], offsetNodenetBoot)
	rack.BootAddresses[3] = addToIP(menu.Network.Bastion.IP, rackIdx, 32)

	rack.BootSystemdAddresses = make([]*net.IPNet, 3)
	rack.BootSystemdAddresses[0] = rack.BootAddresses[0]
	rack.BootSystemdAddresses[1] = addToIPNet(rack.nodeNetworks[1], offsetNodenetToR)
	rack.BootSystemdAddresses[2] = addToIPNet(rack.nodeNetworks[2], offsetNodenetToR)
}

func constructToRAddresses(rack *Rack, rackIdx int, menu *Menu, bases [][]net.IP) {
	rack.ToR1SpineAddresses = make([]*net.IPNet, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR1SpineAddresses[spineIdx] = addToIP(bases[spineIdx][rackIdx], 1, 31)
	}
	rack.ToR1NodeAddress = addToIPNet(rack.nodeNetworks[1], offsetNodenetToR)
	rack.ToR1NodeInterface = fmt.Sprintf("eth%d", menu.Inventory.Spine)

	rack.ToR2SpineAddresses = make([]*net.IPNet, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR2SpineAddresses[spineIdx] = addToIP(bases[spineIdx][rackIdx], 3, 31)
	}
	rack.ToR2NodeAddress = addToIPNet(rack.nodeNetworks[2], offsetNodenetToR)
	rack.ToR2NodeInterface = fmt.Sprintf("eth%d", menu.Inventory.Spine)
}

func addToIPNet(netAddr *net.IPNet, offset int) *net.IPNet {
	ipInt := netutil.IP4ToInt(netAddr.IP) + uint32(offset)
	ip4 := netutil.IntToIP4(ipInt)
	mask := netAddr.Mask
	return &net.IPNet{IP: ip4, Mask: mask}
}

func addToIP(netIP net.IP, offset int, prefixSize int) *net.IPNet {
	ipInt := netutil.IP4ToInt(netIP) + uint32(offset)
	ip4 := netutil.IntToIP4(ipInt)
	mask := net.CIDRMask(prefixSize, 32)
	return &net.IPNet{IP: ip4, Mask: mask}
}

func makeNodeNetwork(base *net.IPNet, nodeIdx int) *net.IPNet {
	mask := base.Mask
	prefixSize, _ := mask.Size()
	offset := 1 << uint(32-prefixSize) * nodeIdx
	ipInt := netutil.IP4ToInt(base.IP) + uint32(offset)
	ip4 := netutil.IntToIP4(ipInt)
	return &net.IPNet{IP: ip4, Mask: mask}
}
