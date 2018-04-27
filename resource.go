package menu

import "net"

type NodeType int

const (
	BootNode NodeType = iota
	CSNode
	SSNode
)

type NetworkMenu struct {
	ASNBase      int
	External     *net.IPNet
	SpineTor     net.IP
	Node         *net.IPNet
	Bastion      *net.IPNet
	LoadBalancer *net.IPNet
	Ingress      *net.IPNet
}

type InventoryMenu struct {
	Spine int
	Rack  []RackMenu
}

type RackMenu struct {
	CS int
	SS int
}

type NodeMenu struct {
	Type   NodeType
	CPU    int
	Memory string
}

type AccountMenu struct {
	UserName string
	Password string
}

type Menu struct {
	Network   *NetworkMenu
	Inventory *InventoryMenu
	Nodes     []*NodeMenu
	Account   *AccountMenu
}
