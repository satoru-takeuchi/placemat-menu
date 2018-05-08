{
  Passwd(): {
    users: [
      {
        name: "{{.Account.Name}}",
        passwordHash: "{{.Account.PasswordHash}}",
        groups: [
          "sudo",
          "docker"
        ],
      }
    ]
  },

  Storage(hostname): {
    files: [
      {
        filesystem: "root",
        path: "/etc/hostname",
        contents: {
          source: "data:," + hostname,
        },
        mode: 420,
      },
      {
        filesystem: "root",
        path: "/etc/systemd/resolved.conf",
        contents: {
          source: "data:,%5BResolve%5D%0ADNS%3D8.8.8.8%0ADNS%3D8.8.4.4%0A",
        },
        mode: 420,
      },
    ],
  },

  local DummyNetworkUnits(name, address) = [
    {
      name: "10-%s.netdev" % name,
      contents: |||
        [NetDev]
        Name=%s
        Kind=dummy
      ||| % name,
    },
    {
      name: "10-%s.network" % name,
      contents: |||
        [Match]
        Name=%s

        [Network]
        Address=%s
      ||| % [name, address],
    },
  ],

  local EthNetworkUnits(addresses) = [
    {
      name: "10-eth%d.network" % i,
      contents: |||
        [Match]
        Name=eth%d

        [Network]
        LLDP=true
        EmitLLDP=nearest-bridge
        Address=%s
      ||| % [i, addresses[i]],
    }
    for i in std.range(0, std.length(addresses)-1)
  ],

  RouterNetwork(addresses): {
    units: EthNetworkUnits(addresses),
  },

  VMNetwork(addr0, addr1, addr2): {
    units: DummyNetworkUnits("node0", addr0) + EthNetworkUnits([addr1, addr2]),
  },

  BootServerNetwork(addr0, addr1, addr2, addr3): {
    units: DummyNetworkUnits("node0", addr0) + EthNetworkUnits([addr1, addr2]) + DummyNetworkUnits("bastion", addr3),
  },

  ExtVMNetwork(addr): {
    units: EthNetworkUnits([addr]),
  },

  Systemd(addresses): {
    local get_addresses(addresses) =
       if (std.length(addresses) > 0) then
           "/usr/bin/ip route add 0.0.0.0/0 src %s nexthop via %s dev eth0 nexthop via %s dev eth1" % [addresses[0], addresses[1], addresses[2]]
       else
           "/bin/true",
    local setup_route = |||
       [Unit]
       After=network.target

       [Service]
       Type=oneshot
       ExecStart=%s

       [Install]
       WantedBy=multi-user.target
    |||,
    units: [
      {
        name: "mnt-containers.mount",
        contents: |||
          [Unit]
          Before=local-fs.target

          [Mount]
          What=/dev/vdb1
          Where=/mnt/containers
          Type=vfat
          Options=ro
        |||,
      },
      {
        name: "rkt-fetch.service",
        contents: |||
          [Unit]
          After=mnt-containers.mount
          Requires=mnt-containers.mount

          [Service]
          Type=oneshot
          RemainAfterExit=yes
          ExecStart=/bin/sh /mnt/containers/rkt-fetch
        |||
      },
      {
        name: "mnt-bird.mount",
        enabled: true,
        contents: |||
          [Unit]
          Before=local-fs.target

          [Mount]
          What=/dev/vdc1
          Where=/mnt/bird
          Type=vfat
          Options=ro

          [Install]
          WantedBy=local-fs.target
        |||,
      },
      {
        name: "copy-bird-conf.service",
        contents: |||
          [Unit]
          After=mnt-bird.mount
          ConditionPathExists=!/etc/bird

          [Service]
          Type=oneshot
          ExecStart=/usr/bin/cp -r /mnt/bird /etc/bird
          RemainAfterExit=yes
        |||,
      },
      {
        name: "copy-bashrc.service",
        enabled: true,
        contents: |||
          [Unit]
          After=mnt-containers.mount
          After=usr.mount

          [Service]
          Type=oneshot
          ExecStart=/usr/bin/mount --bind -o ro /mnt/containers/bashrc /usr/share/skel/.bashrc

          [Install]
          WantedBy=multi-user.target
        |||,
      },
      {
        name: "bird.service",
        enabled: true,
        contents: |||
          [Unit]
          Description=bird
          After=copy-bird-conf.service
          Wants=copy-bird-conf.service
          After=rkt-fetch.service
          Requires=rkt-fetch.service

          [Service]
          Slice=machine.slice
          ExecStart=/usr/bin/rkt run \
            --volume run,kind=empty,readOnly=false \
            --volume etc,kind=host,source=/etc/bird,readOnly=true \
            --net=host \
            quay.io/cybozu/bird:2.0 \
              --readonly-rootfs=true \
              --caps-retain=CAP_NET_ADMIN,CAP_NET_BIND_SERVICE,CAP_NET_RAW \
              --name bird \
              --mount volume=run,target=/run/bird \
              --mount volume=etc,target=/etc/bird \
            quay.io/cybozu/ubuntu-debug:18.04 \
              --readonly-rootfs=true \
              --name ubuntu-debug
          KillMode=mixed
          Restart=on-failure
          RestartForceExitStatus=SIGPIPE

          [Install]
          WantedBy=multi-user.target
        |||,
      },
      {
        name: "setup-iptables.service",
        enabled: true,
        contents: |||
          [Unit]
          After=mnt-bird.mount
          ConditionPathExists=/mnt/bird/setup-iptables

          [Service]
          Type=oneshot
          ExecStart=/bin/sh /mnt/bird/setup-iptables

          [Install]
          WantedBy=multi-user.target
        |||,
      },
      {
        name: "setup-route.service",
        enabled: true,
        contents: setup_route % get_addresses(addresses)
      },
      {
        name: "disable-rp-filter.service",
        enabled: true,
        contents: |||
          [Unit]
          After=mnt-bird.mount
          Before=network-pre.target
          Wants=network-pre.target
          ConditionPathExists=/mnt/bird/setup-rp-filter

          [Service]
          Type=oneshot
          ExecStart=/bin/sh /mnt/bird/setup-rp-filter

          [Install]
          WantedBy=multi-user.target
        |||,
      },
    ]
  },
}