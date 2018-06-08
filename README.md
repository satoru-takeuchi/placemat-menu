# placemat-menu

Placemat-menu is an automatic configuration generation tool for [placemat][placemat].

## Network Design

The cluster of the overview is in the following figure.

![Network Design](http://www.plantuml.com/plantuml/png/hPF1IiGm48RlUOgX9pq4d3p1uYBePGNhJKH26tVBran9P-a-lcsIjIarwyLBwNp__y_03Ddqh1sVlbhXJCNQxbi3nIFr33TFbespXcyBq3qqiH9LIwSQYeVpLEiMHZQGEtgJEV_epvrncXkoi4iCr74wQ4lEG3aqN1syN8rrgfTTOnU6VWBujqMbbbTwyOhJrV7kWydXLJMRnQjP35bXgJQX6Ro9UoA6tKY4b59iI_-FQQ5yKQPAUL7Uasxu7zqkHmHPqs1bsFVqYI3kTurKHAxP7rY6EtlGca-M_gmX6bF9hdE2-aN0N0AJXChDM0gv1EOISSRSTT5kvchDSUN7cQibtnXRJ-_j6m00)


The network is designed as a spine-and-leaf architecture.  There is a *core
switches* at the top of the cluster.  The *spine switches* connects core
switches and racks.  It is a L3 switch to routing between inter cluster,
Internet, and other networks.  

The *operation network* is a network for management of the cluster, it is
normally used for administrators or SRE of the cluster.  Users can reach only
boot servers from the operation network.  The *internet network* is other
cluster or service.  It can reach to the cluster via exposed IP addressed
provided by the cluster.  They are Ingress

Each rack is separated as an individual L3 network.  Although, nodes in a rack
can connect as L2 networks, the nodes over racks are able to each nodes via L3
network.  The nodes uses BGP routing to connect each nodes over all clusters.
The node has a virtual (dummy) network to connect inter-rack nodes.  Its
addresses are advertised to switches and nodes by BGP.  The address of the
physical interface are scoped as link-local, they are used for only L2 network.

The rack has two top of rack (ToR) switches to load balancing network traffics
and increase reliability.  Every node in the rack have two network interfaces
named *node1* and *node2* network.  The node1 interface connect to one ToR
switch, and node2 connects to other ToR switch, respectively.  The advertised
network address in the cluster via BGP is called *node0*.  Additionally, the
boot node has *bastion* network interface.  It is also virtual network, which
is advertised to the operation network.  So uses reach the boot servers via
this IP addresses from operation network.

## Usage

    $ placemat-menu -f <source.yml> [-o <output dir>]

## Getting started

Install placemat-menu to your local disk:

    $ make

placemat-menu loads configuration data from YAML file.
Load this and generate the configuration of placemat as follows:

    $ $GOPATH/bin/placemat-menu -f example.yml -o out

Then, start placemat:

    $ cd out
    $ sudo $GOPATH/bin/placemat cluster.yml

## Specification

See [SPEC.md](SPEC.md)

## Development

placemat-menu utilize [statik][statik] to embed files to the built binary (they
are places in `cmd/placemat-menu/public`).  `statik` is a command to generate
embedded data from static files.  Run the following command to install it:

    $ go get https://github.com/rakyll/statik

It is necessary to run the following to update embedded files when the static
files are modified:

    $ go generate ./...

Then build a placemat-menu:

    $ go build ./cmd/placemat-menu

## License

MIT

[placemat]: https://github.com/cybozu-go/placemat
[statik]: https://github.com/rakyll/statik
