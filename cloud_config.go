package menu

import (
	"fmt"
	"io"
	"net"

	yaml "gopkg.in/yaml.v2"
)

// SeedUser presents a user data in seed file
type SeedUser struct {
	Name string `yaml:"name"`
	Sudo string `yaml:"sudo"`
	// PrimaryGroup string `yaml:"primary-group"`
	Groups     string `yaml:"groups"`
	LockPasswd bool   `yaml:"lock_passwd"`
	Passwd     string `yaml:"passwd"`
	Shell      string `yaml:"shell"`
}

// SeedWriteFile presents a written file in seed file
type SeedWriteFile struct {
	Path    string `yaml:"path"`
	Content string `yaml:"content"`
}

// Seed presents a seed file
type Seed struct {
	Hostname  string     `yaml:"hostname,omitempty"`
	Users     []SeedUser `yaml:"users,omitempty"`
	DiskSetup struct {
		DevVdc struct {
			TableType string `yaml:"table_type"`
			Layout    []int  `yaml:"layout"`
			Overwrite bool   `yaml:"overwrite,omitempty"`
		} `yaml:"/dev/vdc"`
	} `yaml:"disk_setup,omitempty"`
	FsSetup []struct {
		Label      string `yaml:"label"`
		Filesystem string `yaml:"filesystem"`
		Device     string `yaml:"device"`
	} `yaml:"fs_setup"`
	Mounts     [][]string      `yaml:"mounts,omitempty"`
	WriteFiles []SeedWriteFile `yaml:"write_files,omitempty"`
	Runcmd     [][]string      `yaml:"runcmd,omitempty"`
}

func seedDummyNetworkUnits(name string, address *net.IPNet) []SeedWriteFile {
	return []SeedWriteFile{
		{
			Path: fmt.Sprintf("/etc/systemd/network/10-%s.netdev", name),
			Content: fmt.Sprintf(`[NetDev]
Name=%s
Kind=dummy
`, name),
		}, {
			Path: fmt.Sprintf("/etc/systemd/network/10-%s.network", name),
			Content: fmt.Sprintf(`[Match]
Name=%s

[Network]
Address=%s
`, name, address),
		},
	}
}

func seedEthNetworkUnits(addresses []*net.IPNet) []SeedWriteFile {
	units := make([]SeedWriteFile, len(addresses))
	for i, addr := range addresses {
		units[i].Path = fmt.Sprintf("/etc/systemd/network/10-eth%d.network", i)
		units[i].Content = fmt.Sprintf(`[Match]
Name=eth%d

[Network]
LLDP=true
EmitLLDP=nearest-bridge

[Address]
Address=%s
Scope=link
`, i, addr)
	}
	return units
}

// ExportSeed exports a seed
func ExportSeed(w io.Writer, account *Account, rack *Rack) error {
	seed := Seed{
		Hostname: rack.Name + "-boot",
		Users: []SeedUser{
			{
				Name:       account.Name,
				Sudo:       "ALL=(ALL) NOPASSWD:ALL",
				Groups:     "users, admin, systemd-journal, rkt",
				LockPasswd: false,
				Passwd:     account.PasswordHash,
				Shell:      "/bin/bash",
			},
		},
	}

	seed.WriteFiles = seedDummyNetworkUnits("node0", rack.BootNode.Node0Address)
	seed.WriteFiles = append(seed.WriteFiles, seedEthNetworkUnits([]*net.IPNet{rack.BootNode.Node1Address, rack.BootNode.Node2Address})...)
	seed.WriteFiles = append(seed.WriteFiles, seedDummyNetworkUnits("bastion", rack.BootNode.BastionAddress)...)

	seed.Runcmd = append(seed.Runcmd, []string{"systemctl", "restart", "systemd-networkd.service"})

	fmt.Fprintln(w, "#cloud-config")
	return yaml.NewEncoder(w).Encode(seed)
}
