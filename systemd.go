package menu

import (
	"fmt"
	"net"
)

func dummyNetdev(name string) string {
	return fmt.Sprintf(`[NetDev]
Name=%s
Kind=dummy
`, name)
}

func namedNetwork(name string, address *net.IPNet) string {
	return fmt.Sprintf(`[Match]
Name=%s

[Network]
Address=%s
`, name, address)
}

func ethNetwork(name string, address *net.IPNet) string {
	return fmt.Sprintf(`[Match]
Name=%s

[Network]
LLDP=true
EmitLLDP=nearest-bridge

[Address]
Address=%s
Scope=link
`, name, address)
}

func copyBirdConfService() string {
	return `[Unit]
After=mnt-bird.mount
ConditionPathExists=!/etc/bird

[Service]
Type=oneshot
ExecStart=/bin/cp -r /mnt/bird /etc/bird
RemainAfterExit=yes
`
}

func rktFetchService() string {
	return `[Unit]
After=mnt-containers.mount
Requires=mnt-containers.mount

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/sh /mnt/containers/rkt-fetch
`
}

func birdService() string {
	return `[Unit]
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
`
}
