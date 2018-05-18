package main

import (
	"fmt"

	"github.com/cybozu-go/placemat-menu"
	placemat "github.com/cybozu-go/placemat/yaml"
)

const (
	dockerImageBird    = "docker://quay.io/cybozu/bird:2.0"
	dockerImageDebug   = "docker://quay.io/cybozu/ubuntu-debug:18.04"
	dockerImageDnsmasq = "docker://quay.io/cybozu/dnsmasq:2.79"

	aciBird    = "cybozu-bird-2.0.aci"
	aciDebug   = "cybozu-ubuntu-debug-18.04.aci"
	aciDnsmasq = "cybozu-dnsmasq-2.79.aci"

	qemuImageCoreOS = "https://stable.release.core-os.net/amd64-usr/current/coreos_production_qemu_image.img.bz2"
)

var birdContainer = placemat.PodAppConfig{
	Name:           "bird",
	Image:          dockerImageBird,
	ReadOnlyRootfs: true,
	Mount: []placemat.PodAppMountConfig{
		{
			Volume: "config",
			Target: "/etc/bird",
		},
		{
			Volume: "run",
			Target: "/run/bird",
		},
	},
	CapsRetain: []string{
		"CAP_NET_ADMIN",
		"CAP_NET_BIND_SERVICE",
		"CAP_NET_RAW",
	},
}

var debugContainer = placemat.PodAppConfig{
	Name:           "debug",
	Image:          dockerImageDebug,
	ReadOnlyRootfs: true,
}

type cluster struct {
	networks    []*placemat.NetworkConfig
	images      []*placemat.ImageConfig
	dataFolders []*placemat.DataFolderConfig
	pods        []*placemat.PodConfig
	nodes       []*placemat.NodeConfig
}

func generateCluster(ta *menu.TemplateArgs) *cluster {
	cluster := new(cluster)

	cluster.appendExternalNetwork(ta)

	cluster.appendSpineToRackNetwork(ta)

	cluster.appendRackNetwork(ta)

	cluster.appendCoreOSImage()

	cluster.appendCommonDataFolder()

	cluster.appendSpineDataFolder(ta)

	cluster.appendRackDataFolder(ta)

	cluster.appendExtVMDataFolder()

	cluster.appendSpinePod(ta)

	cluster.appendToRPods(ta)

	cluster.appendNodes(ta)

	return cluster
}

func coreOSNode(rackName, rackShortName, nodeName string, resource *menu.VMResource) *placemat.NodeConfig {

	return &placemat.NodeConfig{
		Kind: "Node",
		Name: fmt.Sprintf("%s-%s", rackName, nodeName),
		Spec: placemat.NodeSpec{
			Interfaces: []string{
				fmt.Sprintf("%s-node1", rackShortName),
				fmt.Sprintf("%s-node2", rackShortName),
			},
			Volumes: []placemat.NodeVolumeConfig{
				{
					Kind: "image",
					Name: "root",
					Spec: placemat.NodeVolumeSpec{
						Image:       "coreos-image",
						CopyOnWrite: true,
					},
				},
				{
					Kind: "vvfat",
					Name: "common",
					Spec: placemat.NodeVolumeSpec{
						Folder: "common-data",
					},
				},
				{
					Kind: "vvfat",
					Name: "local",
					Spec: placemat.NodeVolumeSpec{
						Folder: fmt.Sprintf("%s-bird-data", rackName),
					},
				},
			},
			IgnitionFile: fmt.Sprintf("%s-%s.ign", rackName, nodeName),
			Resources: placemat.NodeResourceConfig{
				CPU:    fmt.Sprint(resource.CPU),
				Memory: resource.Memory,
			},
		},
	}
}
func (c *cluster) appendNodes(ta *menu.TemplateArgs) {
	for _, rack := range ta.Racks {
		c.nodes = append(c.nodes, coreOSNode(rack.Name, rack.ShortName, "boot", &ta.Boot))

		for _, cs := range rack.CSList {
			c.nodes = append(c.nodes, coreOSNode(rack.Name, rack.ShortName, cs.Name, &ta.CS))
		}
		for _, ss := range rack.SSList {
			c.nodes = append(c.nodes, coreOSNode(rack.Name, rack.ShortName, ss.Name, &ta.SS))
		}
	}
	c.nodes = append(c.nodes, &placemat.NodeConfig{
		Kind: "Node",
		Name: "ext-vm",
		Spec: placemat.NodeSpec{
			Interfaces: []string{"ext-net"},
			Volumes: []placemat.NodeVolumeConfig{
				{
					Kind: "image",
					Name: "root",
					Spec: placemat.NodeVolumeSpec{
						Image:       "coreos-image",
						CopyOnWrite: true,
					},
				},
				{
					Kind: "vvfat",
					Name: "common",
					Spec: placemat.NodeVolumeSpec{
						Folder: "common-data",
					},
				},
				{
					Kind: "vvfat",
					Name: "local",
					Spec: placemat.NodeVolumeSpec{Folder: "ext-vm-data"},
				},
			},
			IgnitionFile: "ext-vm.ign",
			Resources: placemat.NodeResourceConfig{
				CPU:    "2",
				Memory: "1G",
			},
		},
	})
}

func torPod(rackName, rackShortName string, tor menu.ToR, torNumber int, ta *menu.TemplateArgs) *placemat.PodConfig {

	var spineIfs []placemat.PodInterfaceConfig
	for i, spine := range ta.Spines {
		spineIfs = append(spineIfs,
			placemat.PodInterfaceConfig{
				Network:   fmt.Sprintf("%s-to-%s-%d", spine.ShortName, rackShortName, torNumber),
				Addresses: []string{tor.SpineAddresses[i].String()},
			},
		)
	}
	spineIfs = append(spineIfs, placemat.PodInterfaceConfig{
		Network:   fmt.Sprintf("%s-node%d", rackShortName, torNumber),
		Addresses: []string{tor.NodeAddress.String()},
	})

	dhcpRelayArgs := []string{
		"--log-queries",
		"--log-dhcp",
		"--no-daemon",
	}
	for _, r := range ta.Racks {
		dhcpRelayArgs = append(dhcpRelayArgs, "--dhcp-relay")
		dhcpRelayArgs = append(dhcpRelayArgs, tor.NodeAddress.IP.String()+","+r.BootNode.Node0Address.IP.String())
	}

	return &placemat.PodConfig{
		Kind: "Pod",
		Name: fmt.Sprintf("%s-tor%d", rackName, torNumber),
		Spec: placemat.PodSpec{
			Interfaces: spineIfs,
			Volumes: []placemat.PodVolumeConfig{
				{
					Name:     "config",
					Kind:     "host",
					Folder:   fmt.Sprintf("%s-tor%d-data", rackName, torNumber),
					ReadOnly: true,
				},
				{
					Name: "run",
					Kind: "empty",
				},
			},
			Apps: []placemat.PodAppConfig{
				birdContainer,
				debugContainer,
				{
					Name:           "dhcp-relay",
					Image:          dockerImageDnsmasq,
					ReadOnlyRootfs: true,
					CapsRetain: []string{
						"CAP_NET_BIND_SERVICE",
						"CAP_NET_RAW",
						"CAP_NET_BROADCAST",
					},
					Args: dhcpRelayArgs,
				},
			},
		},
	}
}

func (c *cluster) appendToRPods(ta *menu.TemplateArgs) {
	for _, rack := range ta.Racks {
		c.pods = append(c.pods,
			torPod(rack.Name, rack.ShortName, rack.ToR1, 1, ta),
			torPod(rack.Name, rack.ShortName, rack.ToR2, 2, ta),
		)
	}
}

func (c *cluster) appendSpinePod(ta *menu.TemplateArgs) {
	for _, spine := range ta.Spines {
		var rackIfs []placemat.PodInterfaceConfig
		rackIfs = append(rackIfs, placemat.PodInterfaceConfig{
			Network:   "ext-net",
			Addresses: []string{spine.ExtnetAddress.String()},
		})
		for i, rack := range ta.Racks {
			rackIfs = append(rackIfs,
				placemat.PodInterfaceConfig{
					Network:   fmt.Sprintf("%s-to-%s-1", spine.ShortName, rack.ShortName),
					Addresses: []string{spine.ToR1Address(i).String()},
				},
				placemat.PodInterfaceConfig{
					Network:   fmt.Sprintf("%s-to-%s-2", spine.ShortName, rack.ShortName),
					Addresses: []string{spine.ToR2Address(i).String()},
				},
			)
		}

		c.pods = append(c.pods, &placemat.PodConfig{
			Kind: "Pod",
			Name: spine.Name,
			Spec: placemat.PodSpec{
				InitScripts: []string{"setup-iptables"},
				Interfaces:  rackIfs,
				Volumes: []placemat.PodVolumeConfig{
					{
						Name:     "config",
						Kind:     "host",
						Folder:   fmt.Sprintf("%s-data", spine.Name),
						ReadOnly: true,
					},
					{
						Name: "run",
						Kind: "empty",
					},
				},
				Apps: []placemat.PodAppConfig{
					birdContainer,
					debugContainer,
				},
			},
		})
	}
}

func (c *cluster) appendExtVMDataFolder() {
	c.dataFolders = append(c.dataFolders,
		&placemat.DataFolderConfig{
			Kind: "DataFolder",
			Name: "ext-vm-data",
			Spec: placemat.DataFolderSpec{
				Files: []placemat.DataFolderFileConfig{
					{
						Name: "bird.conf",
						File: "bird_vm.conf",
					},
				},
			},
		})
}

func (c *cluster) appendRackDataFolder(ta *menu.TemplateArgs) {
	for _, rack := range ta.Racks {
		c.dataFolders = append(c.dataFolders,
			&placemat.DataFolderConfig{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-tor1-data", rack.Name),
				Spec: placemat.DataFolderSpec{
					Files: []placemat.DataFolderFileConfig{
						{
							Name: "bird.conf",
							File: fmt.Sprintf("bird_%s-tor1.conf", rack.Name),
						},
					},
				},
			},
			&placemat.DataFolderConfig{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-tor2-data", rack.Name),
				Spec: placemat.DataFolderSpec{
					Files: []placemat.DataFolderFileConfig{
						{
							Name: "bird.conf",
							File: fmt.Sprintf("bird_%s-tor2.conf", rack.Name),
						},
					},
				},
			},
			&placemat.DataFolderConfig{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-bird-data", rack.Name),
				Spec: placemat.DataFolderSpec{
					Files: []placemat.DataFolderFileConfig{
						{
							Name: "bird.conf",
							File: fmt.Sprintf("bird_%s-node.conf", rack.Name),
						},
					},
				},
			},
		)
	}
}

func (c *cluster) appendSpineDataFolder(ta *menu.TemplateArgs) {
	for _, spine := range ta.Spines {
		c.dataFolders = append(c.dataFolders,
			&placemat.DataFolderConfig{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-data", spine.Name),
				Spec: placemat.DataFolderSpec{
					Files: []placemat.DataFolderFileConfig{
						{
							Name: "setup-iptables",
							File: "setup-iptables",
						},
						{
							Name: "bird.conf",
							File: fmt.Sprintf("bird_%s.conf", spine.Name),
						},
					},
				},
			})
	}
}

func (c *cluster) appendCommonDataFolder() {
	c.dataFolders = append(c.dataFolders, &placemat.DataFolderConfig{
		Kind: "DataFolder",
		Name: "common-data",
		Spec: placemat.DataFolderSpec{
			Files: []placemat.DataFolderFileConfig{
				{
					Name: "bird.aci",
					File: aciBird,
				},
				{
					Name: "ubuntu-debug.aci",
					File: aciDebug,
				},
				{
					Name: "dnsmasq.aci",
					File: aciDnsmasq,
				},
				{
					Name: "rkt-fetch",
					File: "rkt-fetch",
				},
				{
					Name: "bashrc",
					File: "bashrc",
				},
			},
		},
	})
}

func (c *cluster) appendCoreOSImage() {
	c.images = append(c.images, &placemat.ImageConfig{
		Kind: "Image",
		Name: "coreos-image",
		Spec: placemat.ImageSpec{
			URL:               qemuImageCoreOS,
			CompressionMethod: "bzip2",
		},
	})
}

func (c *cluster) appendRackNetwork(ta *menu.TemplateArgs) {
	for _, rack := range ta.Racks {
		c.networks = append(
			c.networks,
			&placemat.NetworkConfig{
				Kind: "Network",
				Name: fmt.Sprintf("%s-node1", rack.ShortName),
				Spec: placemat.NetworkSpec{
					Internal: true,
				},
			},
			&placemat.NetworkConfig{
				Kind: "Network",
				Name: fmt.Sprintf("%s-node2", rack.ShortName),
				Spec: placemat.NetworkSpec{
					Internal: true,
				},
			},
		)
	}
}

func (c *cluster) appendSpineToRackNetwork(ta *menu.TemplateArgs) {
	for _, spine := range ta.Spines {
		for _, rack := range ta.Racks {
			c.networks = append(
				c.networks,
				&placemat.NetworkConfig{
					Kind: "Network",
					Name: fmt.Sprintf("%s-to-%s-1", spine.ShortName, rack.ShortName),
					Spec: placemat.NetworkSpec{
						Internal: true,
					},
				},
				&placemat.NetworkConfig{
					Kind: "Network",
					Name: fmt.Sprintf("%s-to-%s-2", spine.ShortName, rack.ShortName),
					Spec: placemat.NetworkSpec{
						Internal: true,
					},
				},
			)
		}
	}
}

func (c *cluster) appendExternalNetwork(ta *menu.TemplateArgs) {
	c.networks = append(
		c.networks,
		&placemat.NetworkConfig{
			Kind: "Network",
			Name: "ext-net",
			Spec: placemat.NetworkSpec{
				Internal:  false,
				UseNAT:    true,
				Addresses: []string{ta.Network.External.Host.String()},
			},
		},
	)
}