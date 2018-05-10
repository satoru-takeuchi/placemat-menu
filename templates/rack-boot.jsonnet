{{$rackIdx := .RackIdx -}}
{{$self := index .Args.Racks $rackIdx -}}
local ign = import 'ign.libsonnet';

{
  ignition: { version: ign.Version },
  passwd: ign.Passwd("{{.Args.Account.Name}}", "{{.Args.Account.PasswordHash}}"),
  storage: ign.Storage("{{$self.Name}}-boot"),

  networkd: ign.BootServerNetwork({{range $addr := $self.BootAddresses}}"{{$addr}}",{{end}}),
  systemd: ign.Systemd([{{range $addr := $self.BootSystemdAddresses}}"{{$addr.IP}}",{{end}}]),
}