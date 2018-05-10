local ign = import 'ign.libsonnet';

{
  ignition: { version: ign.Version },
  passwd: ign.Passwd(),
  storage: ign.Storage("{{.Rack.Name}}-{{.Node.Name}}"),
  networkd: ign.VMNetwork({{range $addr := .Node.Addresses}}"{{$addr}}",{{end}}),
  systemd: ign.Systemd([{{range $addr := .Node.SystemdAddresses}}"{{$addr.IP}}",{{end}}]),
}
