package main

import (
	"bufio"
	"errors"
	"io"
	"net"

	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

type networkConfig struct {
	Spec struct {
		ASNBase  int    `yaml:"asn-base"`
		External string `yaml:"external"`
		SpineTor string `yaml:"spine-tor"`
		Node     string `yaml:"node"`
		Exposed  struct {
			Bastion      string `yaml:"bastion"`
			LoadBalancer string `yaml:"loadbalancer"`
			Ingress      string `yaml:"ingress"`
		} `yaml:"exposed"`
	} `yaml:"spec"`
}

type inventoryConfig struct {
	Spec struct {
		Spine int `yaml:"spine"`
		Rack  []struct {
			CS int `yaml:"cs"`
			SS int `yaml:"ss"`
		} `yaml:"rack"`
	} `yaml:"spec"`
}

type nodeConfig struct {
	Type string `yaml:"type"`
	Spec struct {
		CPU    int    `yaml:"cpu"`
		Memory string `yaml:"memory"`
	} `yaml:"spec"`
}

var nodeType = map[string]menu.NodeType{
	"boot": menu.BootNode,
	"cs":   menu.CSNode,
	"ss":   menu.SSNode,
}

type accountConfig struct {
	Spec struct {
		UserName string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"spec"`
}

func unmarshalNetwork(data []byte) (*menu.Network, error) {
	var n networkConfig
	err := yaml.UnmarshalStrict(data, &n)
	if err != nil {
		return nil, err
	}

	var network menu.Network

	network.ASNBase = n.Spec.ASNBase

	_, network.External, err = net.ParseCIDR(n.Spec.External)
	if err != nil {
		return nil, err
	}

	network.SpineTor = net.ParseIP(n.Spec.SpineTor)
	if network.SpineTor == nil {
		return nil, errors.New("Invalid IP address: " + n.Spec.SpineTor)
	}

	_, network.Node, err = net.ParseCIDR(n.Spec.Node)
	if err != nil {
		return nil, err
	}

	network.Bastion = n.Spec.Exposed.Bastion
	network.LoadBalancer = n.Spec.Exposed.LoadBalancer
	network.Ingress = n.Spec.Exposed.Ingress

	return &network, nil
}

func unmarshalInventory(data []byte) (*menu.Inventory, error) {
	var i inventoryConfig
	err := yaml.UnmarshalStrict(data, &i)
	if err != nil {
		return nil, err
	}

	var inventory menu.Inventory

	if !i.Spec.Spine > 0 {
		return nil, errors.New("spine in Inventory must be more than 0")
	}
	inventory.Spine = i.Spec.Spine

	inventory.Rack = []menu.RackMenu{}
	for _, r := range i.Spec.Rack {
		var rack menu.RackMenu
		rack.CS = r.CS
		rack.SS = r.SS
		inventory.Rack = append(inventory.Rack, rack)
	}

	return &inventory, nil
}

func unmarshalNode(data []byte) (*menu.NodeMenu, error) {
	var n nodeConfig
	err := yaml.UnmarshalStrict(data, &n)
	if err != nil {
		return nil, err
	}

	var node menu.NodeMenu

	node.Type, ok = nodeType[n.Type]
	if !ok {
		return nil, errors.New("Unknown node type: " + n.Type)
	}

	if !n.Spec.CPU > 0 {
		return nil, errors.New("cpu in Node must be more than 0")
	}
	node.CPU, err = n.Spec.CPU

	node.Memory = n.Spec.Memory

	return &node, nil
}

func unmarshalAccount(data []byte) (*menu.Account, error) {
	var a accountConfig
	err := yaml.UnmarshalStrict(data, &a)
	if err != nil {
		return nil, err
	}

	var account menu.Account

	if a.Spec.UserName == "" {
		return nil, errors.New("username is empty")
	}
	account.UserName = a.Spec.UserName

	account.Password = a.Spec.Password

	return &account, nil
}

func readYAML(r *bufio.Reader) (*menu.Menu, error) {
	var m menu.Menu
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
		case "Node":
			r, err := unmarshalNode(data)
			if err != nil {
				return nil, err
			}
			m.Nodes = append(m.Nodes, r)
		case "Account":
			r, err := unmarshalAccount(data)
			if err != nil {
				return nil, err
			}
			m.Account = r
		default:
			return nil, errors.New("unknown resource: " + c.Kind)
		}
	}
	return &m, nil
}
