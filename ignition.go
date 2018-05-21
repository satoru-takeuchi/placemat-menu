package menu

import (
	"fmt"
	"net"
)

// IgnitionPasswdUser contains passwd user information
type IgnitionPasswdUser struct {
	Groups       []string `json:"groups"`
	Name         string   `json:"name"`
	PasswordHash string   `json:"passwordHash"`
}

// IgnitionPasswd contains passwd information
type IgnitionPasswd struct {
	Users []IgnitionPasswdUser `json:"users"`
}

// IgnitionSystemdUnit contains systemd unit information
type IgnitionSystemdUnit struct {
	Contents string `json:"contents"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
}

// IgnitionSystemd contains systemd information
type IgnitionSystemd struct {
	Units []IgnitionSystemdUnit `json:"units"`
}

// IgnitionNetworkdUnit contains networkd unit information
type IgnitionNetworkdUnit struct {
	Contents string `json:"contents"`
	Name     string `json:"name"`
}

// IgnitionNetworkd contains networkd information
type IgnitionNetworkd struct {
	Units []IgnitionNetworkdUnit `json:"units"`
}

// IgnitionStorageFile contains storage by file information
type IgnitionStorageFile struct {
	Contents struct {
		Source string `json:"source"`
	} `json:"contents"`
	FileSystem string `json:"filesystem"`
	Mode       int    `json:"mode"`
	Path       string `json:"path"`
}

// IgnitionStorage contains storage information
type IgnitionStorage struct {
	Files []IgnitionStorageFile `json:"files"`
}

// Ignition contains information to generate an ignition file.
type Ignition struct {
	Ignition struct {
		Version string `json:"version"`
	} `json:"ignition"`

	Passwd IgnitionPasswd `json:"passwd"`

	Storage  IgnitionStorage  `json:"storage"`
	Systemd  IgnitionSystemd  `json:"systemd"`
	Networkd IgnitionNetworkd `json:"networkd"`
}

// IgnitionNode is an interface to generate ignition
type IgnitionNode interface {
	Hostname() string
	Networkd() IgnitionNetworkd
	Systemd() IgnitionSystemd
}

func ignDummyNetworkUnits(name string, address *net.IPNet) []IgnitionNetworkdUnit {
	return []IgnitionNetworkdUnit{
		{
			Name:     fmt.Sprintf("10-%s.netdev", name),
			Contents: dummyNetdev(name),
		}, {
			Name:     fmt.Sprintf("10-%s.network", name),
			Contents: namedNetwork(name, address),
		},
	}
}

func ignEthNetworkUnits(addresses []*net.IPNet) []IgnitionNetworkdUnit {
	units := make([]IgnitionNetworkdUnit, len(addresses))
	for i, addr := range addresses {
		units[i].Name = fmt.Sprintf("10-eth%d.network", i)
		units[i].Contents = ethNetwork(fmt.Sprintf("eth%d", i), addr)
	}
	return units
}

func extVMEthNetwork(addresses []*net.IPNet) []IgnitionNetworkdUnit {
	units := make([]IgnitionNetworkdUnit, len(addresses))
	for i, addr := range addresses {
		units[i].Name = fmt.Sprintf("10-eth%d.network", i)
		units[i].Contents = fmt.Sprintf(`[Match]
Name=eth%d

[Network]
LLDP=true
EmitLLDP=nearest-bridge
Address=%s
`, i, addr)
	}
	return units
}

func defaultSystemdUnits() []IgnitionSystemdUnit {
	return []IgnitionSystemdUnit{
		{
			Name: "mnt-containers.mount",
			Contents: `[Unit]
Before=local-fs.target

[Mount]
What=/dev/vdb1
Where=/mnt/containers
Type=vfat
Options=ro
`,
		}, {
			Name:     "rkt-fetch.service",
			Contents: rktFetchService(),
		}, {
			Name:    "mnt-bird.mount",
			Enabled: true,
			Contents: `[Unit]
Before=local-fs.target

[Mount]
What=/dev/vdc1
Where=/mnt/bird
Type=vfat
Options=ro

[Install]
WantedBy=local-fs.target
`,
		}, {
			Name:     "copy-bird-conf.service",
			Contents: copyBirdConfService(),
		}, {
			Name:    "copy-bashrc.service",
			Enabled: true,
			Contents: `[Unit]
After=mnt-containers.mount
After=usr.mount

[Service]
Type=oneshot
ExecStart=/usr/bin/mount --bind -o ro /mnt/containers/bashrc /usr/share/skel/.bashrc

[Install]
WantedBy=multi-user.target
`,
		}, {
			Name:     "bird.service",
			Enabled:  true,
			Contents: birdService(),
		}, {
			Name:    "setup-iptables.service",
			Enabled: true,
			Contents: `[Unit]
After=mnt-bird.mount
ConditionPathExists=/mnt/bird/setup-iptables

[Service]
Type=oneshot
ExecStart=/bin/sh /mnt/bird/setup-iptables

[Install]
WantedBy=multi-user.target
`,
		},
	}
}

func setupRouteUnit(src, tor1addr, tor2addr net.IP) IgnitionSystemdUnit {
	content := fmt.Sprintf(`[Unit]
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/ip route add 0.0.0.0/0 src %s nexthop via %s nexthop via %s

[Install]
WantedBy=multi-user.target
`, src, tor1addr, tor2addr)

	return IgnitionSystemdUnit{
		Name:     "setup-route.service",
		Enabled:  true,
		Contents: content,
	}
}

func nodeSystemd() IgnitionSystemd {
	return IgnitionSystemd{Units: defaultSystemdUnits()}
}

func bootSystemd(src, ip1, ip2 net.IP) IgnitionSystemd {
	return IgnitionSystemd{Units: append(defaultSystemdUnits(), setupRouteUnit(src, ip1, ip2))}
}

// NodeIgnition returns an Ignition by passwd and node
func NodeIgnition(account Account, node IgnitionNode) Ignition {
	ign := Ignition{}
	ign.Ignition.Version = "2.1.0"
	ign.Passwd = IgnitionPasswd{
		Users: []IgnitionPasswdUser{
			{
				[]string{"sudo", "docker"},
				account.Name,
				account.PasswordHash,
			},
		},
	}
	ign.Storage.Files = []IgnitionStorageFile{
		{
			Contents: struct {
				Source string `json:"source"`
			}{
				Source: "data:," + node.Hostname(),
			},
			FileSystem: "root",
			Mode:       420,
			Path:       "/etc/hostname",
		},
		{
			Contents: struct {
				Source string `json:"source"`
			}{
				Source: `data:,%5BResolve%5D%0ADNS%3D8.8.8.8%0ADNS%3D8.8.4.4%0A`,
			},
			FileSystem: "root",
			Mode:       420,
			Path:       "/etc/systemd/resolved.conf",
		},
	}
	ign.Systemd = node.Systemd()
	ign.Networkd = node.Networkd()
	return ign
}

// BootNodeInfo contains boot server in a rack
type BootNodeInfo struct {
	name        string
	bastionAddr *net.IPNet
	node0Addr   *net.IPNet
	node1Addr   *net.IPNet
	node2Addr   *net.IPNet
	ToR1Addr    net.IP
	ToR2Addr    net.IP
}

// Hostname returns hostname
func (b *BootNodeInfo) Hostname() string {
	return b.name
}

// Networkd returns networkd definitions
func (b *BootNodeInfo) Networkd() IgnitionNetworkd {
	units := make([]IgnitionNetworkdUnit, 0)
	units = append(units, ignDummyNetworkUnits("node0", b.node0Addr)...)
	units = append(units, ignEthNetworkUnits([]*net.IPNet{b.node1Addr, b.node2Addr})...)
	units = append(units, ignDummyNetworkUnits("bastion", b.bastionAddr)...)
	return IgnitionNetworkd{Units: units}
}

// Systemd returns systemd definitions
func (b *BootNodeInfo) Systemd() IgnitionSystemd {
	return bootSystemd(b.bastionAddr.IP, b.ToR1Addr, b.ToR2Addr)
}

// CSNodeInfo contains cs/ss server in a rack
type CSNodeInfo struct {
	name      string
	node0Addr *net.IPNet
	node1Addr *net.IPNet
	node2Addr *net.IPNet
}

// Hostname returns hostname
func (b *CSNodeInfo) Hostname() string {
	return b.name
}

// Networkd returns networkd definitions
func (b *CSNodeInfo) Networkd() IgnitionNetworkd {
	units := make([]IgnitionNetworkdUnit, 0)
	units = append(units, ignDummyNetworkUnits("node0", b.node0Addr)...)
	units = append(units, ignEthNetworkUnits([]*net.IPNet{b.node1Addr, b.node2Addr})...)
	return IgnitionNetworkd{Units: units}

}

// Systemd returns systemd definitions
func (b *CSNodeInfo) Systemd() IgnitionSystemd {
	return nodeSystemd()
}

// SSNodeInfo contains cs/ss server in a rack
type SSNodeInfo struct {
	name      string
	node0Addr *net.IPNet
	node1Addr *net.IPNet
	node2Addr *net.IPNet
}

// Hostname returns hostname
func (b *SSNodeInfo) Hostname() string {
	return b.name
}

// Networkd returns networkd definitions
func (b *SSNodeInfo) Networkd() IgnitionNetworkd {
	units := make([]IgnitionNetworkdUnit, 0)
	units = append(units, ignDummyNetworkUnits("node0", b.node0Addr)...)
	units = append(units, ignEthNetworkUnits([]*net.IPNet{b.node1Addr, b.node2Addr})...)
	return IgnitionNetworkd{Units: units}

}

// Systemd returns systemd definitions
func (b *SSNodeInfo) Systemd() IgnitionSystemd {
	return nodeSystemd()
}

// ExtVMNodeInfo contains external network as VM
type ExtVMNodeInfo struct {
	vmAddr *net.IPNet
}

// Hostname returns hostname
func (b *ExtVMNodeInfo) Hostname() string {
	return "ext-vm"
}

// Networkd returns networkd definitions
func (b *ExtVMNodeInfo) Networkd() IgnitionNetworkd {
	units := extVMEthNetwork([]*net.IPNet{b.vmAddr})
	return IgnitionNetworkd{Units: units}

}

// Systemd returns systemd definitions
func (b *ExtVMNodeInfo) Systemd() IgnitionSystemd {
	return nodeSystemd()
}

// BootNodeIgnition returns an Ignition for boot node
func BootNodeIgnition(account Account, rack Rack) Ignition {
	node := &BootNodeInfo{
		name:        rack.Name + "-boot",
		node0Addr:   rack.BootNode.Node0Address,
		node1Addr:   rack.BootNode.Node1Address,
		node2Addr:   rack.BootNode.Node2Address,
		bastionAddr: rack.BootNode.BastionAddress,
		ToR1Addr:    rack.BootNode.ToR1Address.IP,
		ToR2Addr:    rack.BootNode.ToR2Address.IP,
	}
	return NodeIgnition(account, node)
}

// CSNodeIgnition returns an Ignition for cs/ss servers
func CSNodeIgnition(account Account, rack Rack, node Node) Ignition {
	info := &CSNodeInfo{
		name:      rack.Name + "-" + node.Name,
		node0Addr: node.Node0Address,
		node1Addr: node.Node1Address,
		node2Addr: node.Node2Address,
	}
	return NodeIgnition(account, info)
}

// SSNodeIgnition returns an Ignition for cs/ss servers
func SSNodeIgnition(account Account, rack Rack, node Node) Ignition {
	info := &SSNodeInfo{
		name:      rack.Name + "-" + node.Name,
		node0Addr: node.Node0Address,
		node1Addr: node.Node1Address,
		node2Addr: node.Node2Address,
	}
	return NodeIgnition(account, info)
}

// ExtVMIgnition returns an Ignition for ext-vm
func ExtVMIgnition(account Account, extVMAddr *net.IPNet) Ignition {
	node := &ExtVMNodeInfo{
		vmAddr: extVMAddr,
	}
	return NodeIgnition(account, node)
}
