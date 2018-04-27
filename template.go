package menu

import (
	"fmt"

	"github.com/cybozu-go/netutil"
)

// Rack is template args for rack
type Rack struct {
	Name string
}

// Spine is template args for Spine
type Spine struct {
	Name string
}

// TemplateArgs is args for cluster.yml
type TemplateArgs struct {
	Network struct {
		External struct {
			HostVM string
		}
	}
	Racks  []Rack
	Spines []Spine
}

// MenuToTemplateArgs is converter Menu to TemplateArgs
func MenuToTemplateArgs(menu *Menu) (TemplateArgs, error) {
	var templateArgs TemplateArgs

	extnet := netutil.IP4ToInt(menu.Network.External.IP)
	hostvmip := extnet + 1
	hostvmprefix, _ := menu.Network.External.Mask.Size()
	hostvmipnet := fmt.Sprintf("%s/%d", netutil.IntToIP4(hostvmip).String(), hostvmprefix)
	templateArgs.Network.External.HostVM = hostvmipnet

	templateArgs.Racks = make([]Rack, len(menu.Inventory.Rack))
	for index, _ := range menu.Inventory.Rack {
		templateArgs.Racks[index].Name = fmt.Sprintf("rack%d", index)
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for index := 0; index < menu.Inventory.Spine; index++ {
		templateArgs.Spines[index].Name = fmt.Sprintf("spine%d", index+1)
	}

	return templateArgs, nil
}
