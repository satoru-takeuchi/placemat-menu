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

// SeedDiskSetup presents settings of disks
type SeedDiskSetup struct {
	TableType string `yaml:"table_type"`
	Layout    bool   `yaml:"layout"`
	Overwrite bool   `yaml:"overwrite,omitempty"`
}

// SeedFSSetup presents settings of the file system
type SeedFSSetup struct {
	Label      string `yaml:"label"`
	Filesystem string `yaml:"filesystem"`
	Device     string `yaml:"device"`
}

// Seed presents a seed file
type Seed struct {
	Hostname   string                   `yaml:"hostname,omitempty"`
	Users      []SeedUser               `yaml:"users,omitempty"`
	DiskSetup  map[string]SeedDiskSetup `yaml:"disk_setup,omitempty"`
	FsSetup    []SeedFSSetup            `yaml:"fs_setup"`
	Mounts     [][]string               `yaml:"mounts,omitempty"`
	WriteFiles []SeedWriteFile          `yaml:"write_files,omitempty"`
	Runcmd     [][]string               `yaml:"runcmd,omitempty"`
}

func seedDummyNetworkUnits(name string, address *net.IPNet) []SeedWriteFile {
	return []SeedWriteFile{
		{
			Path:    fmt.Sprintf("/etc/systemd/network/10-%s.netdev", name),
			Content: dummyNetdev(name),
		}, {
			Path:    fmt.Sprintf("/etc/systemd/network/10-%s.network", name),
			Content: namedNetwork(name, address),
		},
	}
}

func seedEthNetworkUnits(addresses []*net.IPNet) []SeedWriteFile {
	units := make([]SeedWriteFile, len(addresses))
	for i, addr := range addresses {
		units[i].Path = fmt.Sprintf("/etc/systemd/network/10-eth%d.network", i)
		units[i].Content = ethNetwork(fmt.Sprintf("ens%d", 3+i), addr)
	}
	return units
}

func systemdWriteFiles() []SeedWriteFile {
	return []SeedWriteFile{
		{
			Path:    "/etc/systemd/system/copy-bird-conf.service",
			Content: copyBirdConfService(),
		},
		{
			Path:    "/etc/systemd/system/rkt-fetch.service",
			Content: rktFetchServiceForUbuntu(),
		},
		{
			Path:    "/etc/systemd/system/bird.service",
			Content: birdService(),
		},
	}
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

	seed.Mounts = append(seed.Mounts,
		[]string{"/dev/vdb1", "/mnt/containers", "auto", "defaults,ro"},
		[]string{"/dev/vdc1", "/mnt/bird", "vfat", "defaults,ro"},
		[]string{"/dev/vdd1", "/var/lib/rkt", "btrfs", "defaults"},
	)

	seed.DiskSetup = make(map[string]SeedDiskSetup)
	seed.DiskSetup["/dev/vdd"] = SeedDiskSetup{TableType: "gpt", Layout: true, Overwrite: false}
	seed.FsSetup = append(seed.FsSetup, SeedFSSetup{Label: "rkt", Filesystem: "btrfs", Device: "/dev/vdd1"})

	seed.WriteFiles = seedDummyNetworkUnits("node0", rack.BootNode.Node0Address)
	seed.WriteFiles = append(seed.WriteFiles, seedEthNetworkUnits([]*net.IPNet{rack.BootNode.Node1Address, rack.BootNode.Node2Address})...)
	seed.WriteFiles = append(seed.WriteFiles, seedDummyNetworkUnits("bastion", rack.BootNode.BastionAddress)...)
	seed.WriteFiles = append(seed.WriteFiles, systemdWriteFiles()...)

	seed.Runcmd = append(seed.Runcmd, []string{"systemctl", "restart", "systemd-networkd.service"})
	seed.Runcmd = append(seed.Runcmd, []string{"dpkg", "-i", "/mnt/containers/rkt.deb"})

	_, err := fmt.Fprintln(w, "#cloud-config")
	if err != nil {
		return err
	}
	return yaml.NewEncoder(w).Encode(seed)
}

// ExportNetworkConfig export network-config file used in cloud-init
func ExportNetworkConfig(w io.Writer) error {
	_, err := fmt.Fprintln(w, "version: 2\nethernets: {}")
	return err
}
