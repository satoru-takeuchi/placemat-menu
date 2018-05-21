images = cybozu-bird-2.0.aci cybozu-ubuntu-debug-18.04.aci cybozu-dnsmasq-2.79.aci
packages = rkt_1.30.0-1_amd64.deb

default: start

cybozu-bird-2.0.aci:
	docker2aci docker://quay.io/cybozu/bird:2.0

cybozu-ubuntu-debug-18.04.aci:
	docker2aci docker://quay.io/cybozu/ubuntu-debug:18.04

cybozu-dnsmasq-2.79.aci:
	docker2aci docker://quay.io/cybozu/dnsmasq:2.79

rkt_1.30.0-1_amd64.deb:
	curl -SL -o rkt_1.30.0-1_amd64.deb "https://github.com/rkt/rkt/releases/download/v1.30.0/rkt_1.30.0-1_amd64.deb"

start: $(images) $(packages)
	@echo "Please run: $(GOPATH)/bin/placemat cluster.yml"

clean:
	sudo rm -rf /var/scratch/placemat

all-clean: clean
	rm -f $(images)
	sudo rm -rf $(HOME)/placemat_data

.PHONY: default start clean all-clean
