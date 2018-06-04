package menu

import (
	"fmt"

	"io"

	placemat "github.com/cybozu-go/placemat/yaml"
	yaml "gopkg.in/yaml.v2"
)

const (
	dockerImageBird    = "docker://quay.io/cybozu/bird:2.0"
	dockerImageDebug   = "docker://quay.io/cybozu/ubuntu-debug:18.04"
	dockerImageDev     = "docker://quay.io/cybozu/ubuntu-dev:18.04"
	dockerImageDnsmasq = "docker://quay.io/cybozu/dnsmasq:2.79"
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
	dataFolders []*placemat.DataFolderConfig
	pods        []*placemat.PodConfig
	nodes       []*placemat.NodeConfig
}

// ExportCluster exports a placemat configuration to writer from TemplateArgs
func ExportCluster(w io.Writer, ta *TemplateArgs) error {
	cluster := generateCluster(ta)

	encoder := yaml.NewEncoder(w)
	for _, n := range cluster.networks {
		err := encoder.Encode(n)
		if err != nil {
			return err
		}
	}
	for _, i := range ta.Images {
		err := encoder.Encode(i)
		if err != nil {
			return err
		}
	}
	for _, f := range cluster.dataFolders {
		err := encoder.Encode(f)
		if err != nil {
			return err
		}
	}
	for _, n := range cluster.nodes {
		err := encoder.Encode(n)
		if err != nil {
			return err
		}
	}
	for _, p := range cluster.pods {
		err := encoder.Encode(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateCluster(ta *TemplateArgs) *cluster {
	cluster := new(cluster)

	cluster.appendExternalNetwork(ta)

	cluster.appendCoreNetwork(ta)

	cluster.appendSpineToRackNetwork(ta)

	cluster.appendRackNetwork(ta)

	cluster.appendCoreDataFolder()

	cluster.appendSpineDataFolder(ta)

	cluster.appendRackDataFolder(ta)

	cluster.appendOperationDataFolder()

	cluster.appendCorePod(ta)

	cluster.appendSpinePod(ta)

	cluster.appendToRPods(ta)

	cluster.appendExtPod(ta)

	cluster.appendOperationPod(ta)

	cluster.appendNodes(ta)

	return cluster
}

func (c *cluster) appendOperationPod(ta *TemplateArgs) {
	pod := &placemat.PodConfig{
		Kind: "Pod",
		Name: "operation",
		Spec: placemat.PodSpec{
			InitScripts: []string{"setup-default-gateway-operation"},
			Interfaces: []placemat.PodInterfaceConfig{
				{
					Network:   "core-to-op",
					Addresses: []string{ta.Network.Endpoints.Operation.String()},
				},
			},
			Volumes: []placemat.PodVolumeConfig{
				{
					Name:   "operation",
					Kind:   "host",
					Folder: "operation-data",
				},
			},
			Apps: []placemat.PodAppConfig{
				{
					Name:  "ubuntu",
					Image: dockerImageDev,
					Exec:  "/bin/sleep",
					Args:  []string{"infinity"},
					Mount: []placemat.PodAppMountConfig{
						{
							Volume: "operation",
							Target: "/mnt",
						},
					},
				},
			},
		},
	}

	c.pods = append(c.pods, pod)
}

func (c *cluster) appendExtPod(ta *TemplateArgs) {
	pod := &placemat.PodConfig{
		Kind: "Pod",
		Name: "external",
		Spec: placemat.PodSpec{
			InitScripts: []string{"setup-default-gateway-external"},
			Interfaces: []placemat.PodInterfaceConfig{
				{
					Network:   "core-to-ext",
					Addresses: []string{ta.Network.Endpoints.External.String()},
				},
			},
			Apps: []placemat.PodAppConfig{
				{
					Name:  "ubuntu",
					Image: dockerImageDebug,
					Exec:  "/bin/sleep",
					Args:  []string{"infinity"},
				},
			},
		},
	}

	c.pods = append(c.pods, pod)
}

func bootNode(rackName, rackShortName, nodeName, serial string, resource *VMResource) *placemat.NodeConfig {
	var volumes []placemat.NodeVolumeConfig
	if resource.Image != "" {
		volumes = []placemat.NodeVolumeConfig{
			{
				Kind: "image",
				Name: "root",
				Spec: placemat.NodeVolumeSpec{
					Image:       resource.Image,
					CopyOnWrite: true,
				},
			},
		}
		if resource.CloudInitTemplate != "" {
			volumes = append(volumes,
				placemat.NodeVolumeConfig{
					Kind: "localds",
					Name: "seed",
					Spec: placemat.NodeVolumeSpec{
						UserData: fmt.Sprintf("seed_%s-%s.yml", rackName, nodeName),
					},
				})
		}
	} else {
		volumes = []placemat.NodeVolumeConfig{
			{
				Kind: "raw",
				Name: "root",
				Spec: placemat.NodeVolumeSpec{
					Size: "30G",
				},
			},
		}
	}

	return &placemat.NodeConfig{
		Kind: "Node",
		Name: fmt.Sprintf("%s-%s", rackName, nodeName),
		Spec: placemat.NodeSpec{
			Interfaces: []string{
				fmt.Sprintf("%s-node1", rackShortName),
				fmt.Sprintf("%s-node2", rackShortName),
			},
			Volumes: volumes,
			Resources: placemat.NodeResourceConfig{
				CPU:    fmt.Sprint(resource.CPU),
				Memory: resource.Memory,
			},
			SMBIOS: placemat.SMBIOSConfig{
				SerialNumber: serial,
			},
		},
	}
}

func emptyNode(rackName, rackShortName, nodeName, serial string, resource *VMResource) *placemat.NodeConfig {

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
					Kind: "raw",
					Name: "root",
					Spec: placemat.NodeVolumeSpec{
						Size: "30G",
					},
				},
			},
			Resources: placemat.NodeResourceConfig{
				CPU:    fmt.Sprint(resource.CPU),
				Memory: resource.Memory,
			},
			BIOS: "uefi",
			SMBIOS: placemat.SMBIOSConfig{
				SerialNumber: serial,
			},
		},
	}
}

func (c *cluster) appendNodes(ta *TemplateArgs) {
	for _, rack := range ta.Racks {
		c.nodes = append(c.nodes, bootNode(rack.Name, rack.ShortName, rack.BootNode.Name, rack.BootNode.Serial, &ta.Boot))

		for _, cs := range rack.CSList {
			c.nodes = append(c.nodes, emptyNode(rack.Name, rack.ShortName, cs.Name, cs.Serial, &ta.CS))
		}
		for _, ss := range rack.SSList {
			c.nodes = append(c.nodes, emptyNode(rack.Name, rack.ShortName, ss.Name, ss.Serial, &ta.SS))
		}
	}
}

func torPod(rackName, rackShortName string, tor ToR, torNumber int, ta *TemplateArgs) *placemat.PodConfig {

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
		if r.Name == rackName {
			continue
		}
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

func (c *cluster) appendToRPods(ta *TemplateArgs) {
	for _, rack := range ta.Racks {
		c.pods = append(c.pods,
			torPod(rack.Name, rack.ShortName, rack.ToR1, 1, ta),
			torPod(rack.Name, rack.ShortName, rack.ToR2, 2, ta),
		)
	}
}

func (c *cluster) appendCorePod(ta *TemplateArgs) {
	var interfaces []placemat.PodInterfaceConfig
	interfaces = append(interfaces, placemat.PodInterfaceConfig{
		Network:   "internet",
		Addresses: []string{ta.Core.InternetAddress.String()},
	})
	for i, spine := range ta.Spines {
		interfaces = append(interfaces, placemat.PodInterfaceConfig{
			Network: fmt.Sprintf("core-to-%s", spine.ShortName),
			Addresses: []string{
				ta.Core.SpineAddresses[i].String(),
			},
		})
	}
	interfaces = append(interfaces, placemat.PodInterfaceConfig{
		Network: "core-to-ext",
		Addresses: []string{
			ta.Core.ExternalAddress.String(),
		},
	})
	interfaces = append(interfaces, placemat.PodInterfaceConfig{
		Network: "core-to-op",
		Addresses: []string{
			ta.Core.OperationAddress.String(),
		},
	})
	c.pods = append(c.pods, &placemat.PodConfig{
		Kind: "Pod",
		Name: "core",
		Spec: placemat.PodSpec{
			InitScripts: []string{"setup-iptables"},
			Interfaces:  interfaces,
			Volumes: []placemat.PodVolumeConfig{
				{
					Name:     "config",
					Kind:     "host",
					Folder:   "core-data",
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

func (c *cluster) appendSpinePod(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		var rackIfs []placemat.PodInterfaceConfig

		rackIfs = append(rackIfs,
			placemat.PodInterfaceConfig{
				Network:   fmt.Sprintf("core-to-%s", spine.ShortName),
				Addresses: []string{spine.CoreAddress.String()},
			},
		)
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
				Interfaces: rackIfs,
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

func (c *cluster) appendOperationDataFolder() {
	c.dataFolders = append(c.dataFolders,
		&placemat.DataFolderConfig{
			Kind: "DataFolder",
			Name: "operation-data",
			Spec: placemat.DataFolderSpec{
				Dir: "operation",
			},
		})
}

func (c *cluster) appendRackDataFolder(ta *TemplateArgs) {
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

func (c *cluster) appendCoreDataFolder() {
	c.dataFolders = append(c.dataFolders,
		&placemat.DataFolderConfig{
			Kind: "DataFolder",
			Name: "core-data",
			Spec: placemat.DataFolderSpec{
				Files: []placemat.DataFolderFileConfig{
					{
						Name: "bird.conf",
						File: "bird_core.conf",
					},
				},
			},
		})
}

func (c *cluster) appendSpineDataFolder(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		c.dataFolders = append(c.dataFolders,
			&placemat.DataFolderConfig{
				Kind: "DataFolder",
				Name: fmt.Sprintf("%s-data", spine.Name),
				Spec: placemat.DataFolderSpec{
					Files: []placemat.DataFolderFileConfig{
						{
							Name: "bird.conf",
							File: fmt.Sprintf("bird_%s.conf", spine.Name),
						},
					},
				},
			})
	}
}

func (c *cluster) appendRackNetwork(ta *TemplateArgs) {
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

func (c *cluster) appendSpineToRackNetwork(ta *TemplateArgs) {
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

func (c *cluster) appendExternalNetwork(ta *TemplateArgs) {
	c.networks = append(
		c.networks,
		&placemat.NetworkConfig{
			Kind: "Network",
			Name: "internet",
			Spec: placemat.NetworkSpec{
				Internal:  false,
				UseNAT:    true,
				Addresses: []string{ta.Network.Endpoints.Host.String()},
			},
		},
	)
}

func (c *cluster) appendCoreNetwork(ta *TemplateArgs) {
	for _, spine := range ta.Spines {
		c.networks = append(c.networks, &placemat.NetworkConfig{
			Kind: "Network",
			Name: fmt.Sprintf("core-to-%s", spine.ShortName),
			Spec: placemat.NetworkSpec{
				Internal: true,
			},
		})
	}
	c.networks = append(
		c.networks,
		&placemat.NetworkConfig{
			Kind: "Network",
			Name: "core-to-ext",
			Spec: placemat.NetworkSpec{
				Internal: true,
			},
		},
		&placemat.NetworkConfig{
			Kind: "Network",
			Name: "core-to-op",
			Spec: placemat.NetworkSpec{
				Internal: true,
			},
		},
	)
}
