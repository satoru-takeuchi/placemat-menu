package menu

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/cybozu-go/sabakan"
)

func copyIPAMConfig(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}

func exportJSON(dst string, data interface{}) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func exportDHCPConfig(dst string) error {
	config := sabakan.DHCPConfig{
		GatewayOffset: offsetNodenetToR,
		LeaseMinutes:  60,
	}

	return exportJSON(dst, config)
}

func sabakanMachine(serial string, rack int, role string) sabakan.MachineSpec {
	return sabakan.MachineSpec{
		Serial:     serial,
		Product:    "vm",
		Datacenter: "dc1",
		Rack:       uint(rack),
		Role:       role,
		BMC: sabakan.MachineBMC{
			Type: sabakan.BmcIpmi2,
		},
	}
}

func exportMachinesJSON(dst string, ta *TemplateArgs) error {
	var ms []sabakan.MachineSpec

	for _, rack := range ta.Racks {
		ms = append(ms, sabakanMachine(rack.BootNode.Serial, rack.Index, "boot"))

		for _, cs := range rack.CSList {
			ms = append(ms, sabakanMachine(cs.Serial, rack.Index, "worker"))
		}
		for _, ss := range rack.SSList {
			ms = append(ms, sabakanMachine(ss.Serial, rack.Index, "worker"))
		}
	}

	return exportJSON(dst, ms)
}

// ExportSabakanData exports configuration files for sabakan
func ExportSabakanData(dir string, m *Menu, ta *TemplateArgs) error {
	err := copyIPAMConfig(m.Network.IPAMConfigFile, filepath.Join(dir, "ipam.json"))
	if err != nil {
		return err
	}

	err = exportDHCPConfig(filepath.Join(dir, "dhcp.json"))
	if err != nil {
		return err
	}

	return exportMachinesJSON(filepath.Join(dir, "machines.json"), ta)
}
