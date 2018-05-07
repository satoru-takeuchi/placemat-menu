local ign = import 'ign.libsonnet';

local ignition_version = "2.1.0";

{
  {{range $spine := .Spines -}}
  "{{$spine.Name}}.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("spine"),
    networkd: ign.RouterNetwork([{{range $addr := $spine.Addresses}}"{{$addr}}",{{end}}]),
    systemd: ign.Systemd([]),
  },
  {{end -}}
  {{range $rack := .Racks -}}
  "{{$rack.Name}}-tor1.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-tor1"),
    networkd: ign.RouterNetwork([{{range $addr := $rack.ToR1Addresses}}"{{$addr}}",{{end}}]),
    systemd: ign.Systemd([]),
  },
  "{{$rack.Name}}-tor2.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-tor2"),
    networkd: ign.RouterNetwork([{{range $addr := $rack.ToR2Addresses}}"{{$addr}}",{{end}}]),
    systemd: ign.Systemd([]),
  },
  "{{$rack.Name}}-boot.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-boot"),
    networkd: ign.BootServerNetwork({{range $addr := $rack.BootAddresses}}"{{$addr}}",{{end}}),
    systemd: ign.Systemd([{{range $addr := $rack.BootSystemdAddresses}}"{{$addr}}",{{end}}]),
  },
  {{range $cs := $rack.CSList -}}
  "{{$rack.Name}}-{{$cs.Name}}.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-{{$cs.Name}}"),
    networkd: ign.VMNetwork({{range $addr := $cs.Addresses}}"{{$addr}}",{{end}}),
    systemd: ign.Systemd([{{range $addr := $cs.SystemdAddresses}}"{{$addr}}",{{end}}]),
  },
  {{end -}}
  {{range $ss := $rack.SSList -}}
  "{{$rack.Name}}-{{$ss.Name}}.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-{{$ss.Name}}"),
    networkd: ign.VMNetwork({{range $addr := $ss.Addresses}}"{{$addr}}",{{end}}),
    systemd: ign.Systemd([{{range $addr := $ss.SystemdAddresses}}"{{$addr}}",{{end}}]),
  },
  {{end -}}
  {{end -}}
  "forest.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("forest"),
    networkd: ign.ForestNetwork("{{.Network.External.VM}}"),
    systemd: ign.Systemd([]),
  },
}
