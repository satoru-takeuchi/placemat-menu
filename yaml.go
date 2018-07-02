package menu

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/cybozu-go/placemat"
	"github.com/cybozu-go/sabakan"
	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

type networkConfig struct {
	Spec struct {
		IPAMConfig    string `yaml:"ipam-config"`
		ASNBase       int    `yaml:"asn-base"`
		Internet      string `yaml:"internet"`
		SpineTor      string `yaml:"spine-tor"`
		CoreSpine     string `yaml:"core-spine"`
		CoreExternal  string `yaml:"core-external"`
		CoreOperation string `yaml:"core-operation"`
		Exposed       struct {
			Bastion      string `yaml:"bastion"`
			LoadBalancer string `yaml:"loadbalancer"`
			Ingress      string `yaml:"ingress"`
		} `yaml:"exposed"`
	} `yaml:"spec"`
}

type inventoryConfig struct {
	Spec struct {
		ClusterID string `yaml:"cluster-id"`
		Spine     int    `yaml:"spine"`
		Rack      []struct {
			CS int `yaml:"cs"`
			SS int `yaml:"ss"`
		} `yaml:"rack"`
	} `yaml:"spec"`
}

type imageSpec = placemat.ImageSpec

type nodeConfig struct {
	Type string `yaml:"type"`
	Spec struct {
		CPU               int    `yaml:"cpu"`
		Memory            string `yaml:"memory"`
		Image             string `yaml:"image"`
		UEFI              bool   `yaml:"uefi"`
		CloudInitTemplate string `yaml:"cloud-init-template"`
	} `yaml:"spec"`
}

var nodeType = map[string]NodeType{
	"boot": BootNode,
	"cs":   CSNode,
	"ss":   SSNode,
}

func parseNetworkCIDR(s string) (net.IP, *net.IPNet, error) {
	ip, network, err := net.ParseCIDR(s)
	if err != nil {
		return nil, nil, err
	}
	if !ip.Equal(network.IP) {
		return nil, nil, errors.New("Host part of network address must be 0: " + s)
	}
	return ip, network, nil
}

func unmarshalNetwork(data []byte) (*NetworkMenu, error) {
	var n networkConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}

	var network NetworkMenu

	network.IPAMConfigFile = n.Spec.IPAMConfig
	f, err := os.Open(network.IPAMConfigFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ic sabakan.IPAMConfig
	err = json.NewDecoder(f).Decode(&ic)
	if err != nil {
		return nil, err
	}
	if ic.NodeIPPerNode != torPerRack+1 {
		return nil, fmt.Errorf("node-ip-per-node in IPAM config must be %d", torPerRack+1)
	}
	if ic.NodeIndexOffset != offsetNodenetBoot {
		return nil, fmt.Errorf("node-index-offset in IPAM config must be %d", offsetNodenetBoot)
	}
	network.NodeBase, _, err = parseNetworkCIDR(ic.NodeIPv4Pool)
	if err != nil {
		return nil, err
	}
	network.NodeRangeSize = int(ic.NodeRangeSize)
	network.NodeRangeMask = int(ic.NodeRangeMask)
	_, network.BMC, err = parseNetworkCIDR(ic.BMCIPv4Pool)
	if err != nil {
		return nil, err
	}

	network.ASNBase = n.Spec.ASNBase

	_, network.Internet, err = parseNetworkCIDR(n.Spec.Internet)
	if err != nil {
		return nil, err
	}

	_, network.CoreOperation, err = parseNetworkCIDR(n.Spec.CoreOperation)
	if err != nil {
		return nil, err
	}
	_, network.CoreSpine, err = parseNetworkCIDR(n.Spec.CoreSpine)
	if err != nil {
		return nil, err
	}
	_, network.CoreExternal, err = parseNetworkCIDR(n.Spec.CoreExternal)
	if err != nil {
		return nil, err
	}
	network.SpineTor = net.ParseIP(n.Spec.SpineTor)
	if network.SpineTor == nil {
		return nil, errors.New("Invalid IP address: " + n.Spec.SpineTor)
	}

	_, network.Bastion, err = parseNetworkCIDR(n.Spec.Exposed.Bastion)
	if err != nil {
		return nil, err
	}
	_, network.LoadBalancer, err = parseNetworkCIDR(n.Spec.Exposed.LoadBalancer)
	if err != nil {
		return nil, err
	}
	_, network.Ingress, err = parseNetworkCIDR(n.Spec.Exposed.Ingress)
	if err != nil {
		return nil, err
	}

	return &network, nil
}

func unmarshalInventory(data []byte) (*InventoryMenu, error) {
	var i inventoryConfig
	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return nil, err
	}

	var inventory InventoryMenu

	if i.Spec.ClusterID == "" {
		return nil, errors.New("cluster-id is empty")
	}
	inventory.ClusterID = i.Spec.ClusterID

	if !(i.Spec.Spine > 0) {
		return nil, errors.New("spine in Inventory must be more than 0")
	}
	inventory.Spine = i.Spec.Spine

	inventory.Rack = []RackMenu{}
	for _, r := range i.Spec.Rack {
		var rack RackMenu
		rack.CS = r.CS
		rack.SS = r.SS
		inventory.Rack = append(inventory.Rack, rack)
	}

	return &inventory, nil
}

func unmarshalImage(data []byte) (*imageSpec, error) {
	var i imageSpec
	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func unmarshalNode(data []byte) (*NodeMenu, error) {
	var n nodeConfig
	err := yaml.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}

	var node NodeMenu

	nodetype, ok := nodeType[n.Type]
	if !ok {
		return nil, errors.New("Unknown node type: " + n.Type)
	}
	node.Type = nodetype

	if !(n.Spec.CPU > 0) {
		return nil, errors.New("cpu in Node must be more than 0")
	}
	node.CPU = n.Spec.CPU

	node.Memory = n.Spec.Memory
	node.Image = n.Spec.Image
	node.UEFI = n.Spec.UEFI
	node.CloudInitTemplate = n.Spec.CloudInitTemplate

	return &node, nil
}

// ReadYAML read placemat-menu resource files
func ReadYAML(r *bufio.Reader) (*Menu, error) {
	var m Menu
	var c baseConfig
	y := k8sYaml.NewYAMLReader(r)
	for {
		data, err := y.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return nil, err
		}

		switch c.Kind {
		case "Network":
			r, err := unmarshalNetwork(data)
			if err != nil {
				return nil, err
			}
			m.Network = r
		case "Inventory":
			r, err := unmarshalInventory(data)
			if err != nil {
				return nil, err
			}
			m.Inventory = r
		case "Image":
			r, err := unmarshalImage(data)
			if err != nil {
				return nil, err
			}
			m.Images = append(m.Images, r)
		case "Node":
			r, err := unmarshalNode(data)
			if err != nil {
				return nil, err
			}
			m.Nodes = append(m.Nodes, r)
		default:
			return nil, errors.New("unknown resource: " + c.Kind)
		}
	}
	return &m, nil
}
