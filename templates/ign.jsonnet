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
  "rack0-tor1.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("rack0-tor1"),
    networkd: ign.RouterNetwork(["10.0.1.1/31", "10.69.0.65/26"]),
    systemd: ign.Systemd([]),
  },
  "rack0-tor2.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("rack0-tor2"),
    networkd: ign.RouterNetwork(["10.0.1.3/31", "10.69.0.129/26"]),
    systemd: ign.Systemd([]),
  },
  "rack0-boot.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("rack0-boot"),
    networkd: ign.BootServerNetwork("10.69.0.3/32", "10.69.0.67/26", "10.69.0.131/26", "10.72.48.0/32"),
    systemd: ign.Systemd(["10.69.0.3", "10.69.0.65", "10.69.0.129"]),
  },
  "rack0-cs1.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("rack0-cs1"),
    networkd: ign.VMNetwork("10.69.0.4/32", "10.69.0.68/26", "10.69.0.132/26"),
    systemd: ign.Systemd(["10.69.0.4", "10.69.0.65", "10.69.0.129"]),
  },
  "rack0-cs2.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("rack0-cs2"),
    networkd: ign.VMNetwork("10.69.0.5/32", "10.69.0.69/26", "10.69.0.133/26"),
    systemd: ign.Systemd(["10.69.0.5", "10.69.0.65", "10.69.0.129"]),
  },
  "forest.ign": {
    ignition: { version: ignition_version },
    passwd: ign.Passwd(),
    storage: ign.Storage("forest"),
    networkd: ign.ForestNetwork("10.0.2.1/32", "10.0.0.3/24"),
    systemd: ign.Systemd([]),
  },
}