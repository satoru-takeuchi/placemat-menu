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

// Ignition contains an igniration information
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

func dummyNetworkUnits(name string, address *net.IPNet) []IgnitionNetworkdUnit {
	return []IgnitionNetworkdUnit{
		{
			Name: fmt.Sprintf("10-%s.netdev", name),
			Contents: fmt.Sprintf(`[NetDev]
Name=%s
Kind=dummy
`, name),
		}, {
			Name: fmt.Sprintf("10-%s.network", name),
			Contents: fmt.Sprintf(`[Match]
Name=%s

[Network]
Address=%s
`, name, address),
		},
	}
}

func ethNetworkUnits(addresses []*net.IPNet) []IgnitionNetworkdUnit {
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

func defaultSystemd(addresses []net.IP) IgnitionSystemd {
	getAddressesCommandLine := func(addresses []net.IP) string {
		if len(addresses) == 0 {
			return "/bin/true"
		}
		return fmt.Sprintf("/usr/bin/ip route add 0.0.0.0/0 src %s nexthop via %s dev eth0 nexthop via %s dev eth1", addresses[0], addresses[1], addresses[2])
	}

	setupRouteSystemdContent := func(cmd string) string {
		return fmt.Sprintf(`[Unit]
After=network.target

[Service]
Type=oneshot
ExecStart=%s

[Install]
WantedBy=multi-user.target
`, cmd)
	}

	units := []IgnitionSystemdUnit{
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
			Name: "rkt-fetch.service",
			Contents: `[Unit]
After=mnt-containers.mount
Requires=mnt-containers.mount

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/sh /mnt/containers/rkt-fetch
`,
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
			Name: "copy-bird-conf.service",
			Contents: `[Unit]
After=mnt-bird.mount
ConditionPathExists=!/etc/bird

[Service]
Type=oneshot
ExecStart=/usr/bin/cp -r /mnt/bird /etc/bird
RemainAfterExit=yes
`,
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
			Name:    "bird.service",
			Enabled: true,
			Contents: `[Unit]
Description=bird
After=copy-bird-conf.service
Wants=copy-bird-conf.service
After=rkt-fetch.service
Requires=rkt-fetch.service

[Service]
Slice=machine.slice
ExecStart=/usr/bin/rkt run \
  --volume run,kind=empty,readOnly=false \
  --volume etc,kind=host,source=/etc/bird,readOnly=true \
  --net=host \
  quay.io/cybozu/bird:2.0 \
    --readonly-rootfs=true \
    --caps-retain=CAP_NET_ADMIN,CAP_NET_BIND_SERVICE,CAP_NET_RAW \
    --name bird \
    --mount volume=run,target=/run/bird \
    --mount volume=etc,target=/etc/bird \
  quay.io/cybozu/ubuntu-debug:18.04 \
    --readonly-rootfs=true \
    --name ubuntu-debug
KillMode=mixed
Restart=on-failure
RestartForceExitStatus=SIGPIPE

[Install]
WantedBy=multi-user.target
`,
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
		}, {
			Name:     "setup-route.service",
			Enabled:  true,
			Contents: setupRouteSystemdContent(getAddressesCommandLine(addresses)),
		}, {
			Name:    "disable-rp-filter.service",
			Enabled: true,
			Contents: `[Unit]
After=mnt-bird.mount
Before=network-pre.target
Wants=network-pre.target
ConditionPathExists=/mnt/bird/setup-rp-filter

[Service]
Type=oneshot
ExecStart=/bin/sh /mnt/bird/setup-rp-filter

[Install]
WantedBy=multi-user.target
`,
		},
	}
	return IgnitionSystemd{Units: units}
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
	name             string
	bastionAddr      *net.IPNet
	node0Addr        *net.IPNet
	node1Addr        *net.IPNet
	node2Addr        *net.IPNet
	node0SystemdAddr net.IP
	node1SystemdAddr net.IP
	node2SystemdAddr net.IP
}

// Hostname returns hostname
func (b *BootNodeInfo) Hostname() string {
	return b.name
}

// Networkd returns networkd definitions
func (b *BootNodeInfo) Networkd() IgnitionNetworkd {
	units := make([]IgnitionNetworkdUnit, 0)
	units = append(units, dummyNetworkUnits("node0", b.node0Addr)...)
	units = append(units, ethNetworkUnits([]*net.IPNet{b.node1Addr, b.node2Addr})...)
	units = append(units, dummyNetworkUnits("bastion", b.bastionAddr)...)
	return IgnitionNetworkd{Units: units}
}

// Systemd returns systemd definitions
func (b *BootNodeInfo) Systemd() IgnitionSystemd {
	return defaultSystemd([]net.IP{b.node0SystemdAddr, b.node1SystemdAddr, b.node2SystemdAddr})
}

// CSNodeInfo contains cs/ss server in a rack
type CSNodeInfo struct {
	name             string
	node0Addr        *net.IPNet
	node1Addr        *net.IPNet
	node2Addr        *net.IPNet
	node0SystemdAddr net.IP
	node1SystemdAddr net.IP
	node2SystemdAddr net.IP
}

// Hostname returns hostname
func (b *CSNodeInfo) Hostname() string {
	return b.name
}

// Networkd returns networkd definitions
func (b *CSNodeInfo) Networkd() IgnitionNetworkd {
	units := make([]IgnitionNetworkdUnit, 0)
	units = append(units, dummyNetworkUnits("node0", b.node0Addr)...)
	units = append(units, ethNetworkUnits([]*net.IPNet{b.node1Addr, b.node2Addr})...)
	return IgnitionNetworkd{Units: units}

}

// Systemd returns systemd definitions
func (b *CSNodeInfo) Systemd() IgnitionSystemd {
	return defaultSystemd([]net.IP{b.node0SystemdAddr, b.node1SystemdAddr, b.node2SystemdAddr})
}

// SSNodeInfo contains cs/ss server in a rack
type SSNodeInfo struct {
	name             string
	node0Addr        *net.IPNet
	node1Addr        *net.IPNet
	node2Addr        *net.IPNet
	node0SystemdAddr net.IP
	node1SystemdAddr net.IP
	node2SystemdAddr net.IP
}

// Hostname returns hostname
func (b *SSNodeInfo) Hostname() string {
	return b.name
}

// Networkd returns networkd definitions
func (b *SSNodeInfo) Networkd() IgnitionNetworkd {
	units := make([]IgnitionNetworkdUnit, 0)
	units = append(units, dummyNetworkUnits("node0", b.node0Addr)...)
	units = append(units, ethNetworkUnits([]*net.IPNet{b.node1Addr, b.node2Addr})...)
	return IgnitionNetworkd{Units: units}

}

// Systemd returns systemd definitions
func (b *SSNodeInfo) Systemd() IgnitionSystemd {
	return defaultSystemd([]net.IP{b.node0SystemdAddr, b.node1SystemdAddr, b.node2SystemdAddr})
}

// ExtVMNodeInfo contains external network as VM
type ExtVMNodeInfo struct {
	vmAddr *net.IPNet
}

// Hostname returns hostname
func (b *ExtVMNodeInfo) Hostname() string {
	return "forest"
}

// Networkd returns networkd definitions
func (b *ExtVMNodeInfo) Networkd() IgnitionNetworkd {
	units := ethNetworkUnits([]*net.IPNet{b.vmAddr})
	return IgnitionNetworkd{Units: units}

}

// Systemd returns systemd definitions
func (b *ExtVMNodeInfo) Systemd() IgnitionSystemd {
	return defaultSystemd([]net.IP{})
}

// BootNodeIgnition returns an Ignition for boot node
func BootNodeIgnition(account Account, rack Rack) Ignition {
	node := &BootNodeInfo{
		name:             rack.Name + "-boot",
		node0Addr:        rack.BootAddresses[0],
		node1Addr:        rack.BootAddresses[1],
		node2Addr:        rack.BootAddresses[2],
		bastionAddr:      rack.BootAddresses[3],
		node0SystemdAddr: rack.BootAddresses[0].IP,
		node1SystemdAddr: rack.BootSystemdAddresses[1].IP,
		node2SystemdAddr: rack.BootSystemdAddresses[2].IP,
	}
	return NodeIgnition(account, node)
}

// CSNodeIgnition returns an Ignition for cs/ss servers
func CSNodeIgnition(account Account, rack Rack, node Node) Ignition {
	info := &CSNodeInfo{
		name:             rack.Name + "-" + node.Name,
		node0Addr:        node.Addresses[0],
		node1Addr:        node.Addresses[1],
		node2Addr:        node.Addresses[2],
		node0SystemdAddr: node.Addresses[0].IP,
		node1SystemdAddr: node.SystemdAddresses[1].IP,
		node2SystemdAddr: node.SystemdAddresses[2].IP,
	}
	return NodeIgnition(account, info)
}

// SSNodeIgnition returns an Ignition for cs/ss servers
func SSNodeIgnition(account Account, rack Rack, node Node) Ignition {
	info := &SSNodeInfo{
		name:             rack.Name + "-" + node.Name,
		node0Addr:        node.Addresses[0],
		node1Addr:        node.Addresses[1],
		node2Addr:        node.Addresses[2],
		node0SystemdAddr: node.Addresses[0].IP,
		node1SystemdAddr: node.SystemdAddresses[1].IP,
		node2SystemdAddr: node.SystemdAddresses[2].IP,
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
