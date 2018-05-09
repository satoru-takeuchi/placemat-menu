local ign = import 'ign.libsonnet';

local ignition_version = "2.1.0";

{
  {{range $spine := .Spines -}}
  "{{$spine.Name}}.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$spine.Name}}"),
    networkd: ign.RouterNetwork(["{{$spine.ExtnetAddress}}",{{range $addr := $spine.ToRAddresses}}"{{$addr}}",{{end}}]),
    systemd: ign.Systemd([]),
  },
  {{end -}}
  {{range $rack := .Racks -}}
  "{{$rack.Name}}-tor1.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-tor1"),
    networkd: ign.RouterNetwork([{{range $addr := $rack.ToR1SpineAddresses}}"{{$addr}}",{{end}}"{{$rack.ToR1NodeAddress}}"]),
    systemd: ign.Systemd([]),
  },
  "{{$rack.Name}}-tor2.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-tor2"),
    networkd: ign.RouterNetwork([{{range $addr := $rack.ToR2SpineAddresses}}"{{$addr}}",{{end}}"{{$rack.ToR2NodeAddress}}"]),
    systemd: ign.Systemd([]),
  },
  "{{$rack.Name}}-boot.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-boot"),
    networkd: ign.BootServerNetwork({{range $addr := $rack.BootAddresses}}"{{$addr}}",{{end}}),
    systemd: ign.Systemd([{{range $addr := $rack.BootSystemdAddresses}}"{{$addr.IP}}",{{end}}]),
  },
  {{range $cs := $rack.CSList -}}
  "{{$rack.Name}}-{{$cs.Name}}.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-{{$cs.Name}}"),
    networkd: ign.VMNetwork({{range $addr := $cs.Addresses}}"{{$addr}}",{{end}}),
    systemd: ign.Systemd([{{range $addr := $cs.SystemdAddresses}}"{{$addr.IP}}",{{end}}]),
  },
  {{end -}}
  {{range $ss := $rack.SSList -}}
  "{{$rack.Name}}-{{$ss.Name}}.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-{{$ss.Name}}"),
    networkd: ign.VMNetwork({{range $addr := $ss.Addresses}}"{{$addr}}",{{end}}),
    systemd: ign.Systemd([{{range $addr := $ss.SystemdAddresses}}"{{$addr.IP}}",{{end}}]),
  },
  {{end -}}
  {{end -}}
  "ext-vm.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("ext-vm"),
    networkd: ign.ExtVMNetwork("{{.Network.External.VM}}"),
    systemd: ign.Systemd([]),
  },
}
