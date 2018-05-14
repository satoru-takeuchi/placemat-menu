images = cybozu-bird-2.0.aci cybozu-ubuntu-debug-18.04.aci

default: start

cybozu-bird-2.0.aci:
	docker2aci docker://quay.io/cybozu/bird:2.0

cybozu-ubuntu-debug-18.04.aci:
	docker2aci docker://quay.io/cybozu/ubuntu-debug:18.04

start: $(images)
	@echo "Please run: $(GOPATH)/bin/placemat cluster.yml"

clean:
	sudo rm -rf /var/scratch/placemat

all-clean: clean
	rm -f $(images)
	sudo rm -rf $(HOME)/placemat_data

.PHONY: default start clean all-clean
