ignitions = ext-vm.ign rack0-boot.ign \
	rack0-cs1.ign rack0-cs2.ign \
	rack1-boot.ign \
	rack1-cs1.ign rack1-cs2.ign \
	rack1-ss1.ign rack1-ss2.ign 
images = cybozu-bird-2.0.aci cybozu-ubuntu-debug-18.04.aci

default: start

jsonnet: $(ignitions)
%.ign: %.jsonnet ign.libsonnet
	jsonnet -o $@ $<
	validate $@

cybozu-bird-2.0.aci:
	docker2aci docker://quay.io/cybozu/bird:2.0

cybozu-ubuntu-debug-18.04.aci:
	docker2aci docker://quay.io/cybozu/ubuntu-debug:18.04

start: $(ignitions) $(images)
	@echo "Please run: $(GOPATH)/bin/placemat cluster.yml"

clean:
	rm -f $(ignitions)
	sudo rm -rf /var/scratch/placemat

all-clean: clean
	rm -f $(images)
	sudo rm -rf $(HOME)/placemat_data

.PHONY: default start clean all-clean
