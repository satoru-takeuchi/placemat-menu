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
  "{{$rack.Name}}-cs1.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-cs1"),
    networkd: ign.VMNetwork("10.69.0.4/32", "10.69.0.68/26", "10.69.0.132/26"),
    systemd: ign.Systemd(["10.69.0.4", "10.69.0.65", "10.69.0.129"]),
  },
  "{{$rack.Name}}-cs2.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("{{$rack.Name}}-cs2"),
    networkd: ign.VMNetwork("10.69.0.5/32", "10.69.0.69/26", "10.69.0.133/26"),
    systemd: ign.Systemd(["10.69.0.5", "10.69.0.65", "10.69.0.129"]),
  },
  {{end -}}
  "forest.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("forest"),
    networkd: ign.ForestNetwork("10.0.2.1/32", "10.0.0.3/24"),
    systemd: ign.Systemd([]),
  },
}
