local ign = import 'ign.libsonnet';

{
  ignition: { version: ign.Version },
  passwd: ign.Passwd("{{.Account.Name}}", "{{.Account.PasswordHash}}"),
  storage: ign.Storage("forest"),
  networkd: ign.ExtVMNetwork("{{.Network.External.VM}}"),
  systemd: ign.Systemd([]),
}