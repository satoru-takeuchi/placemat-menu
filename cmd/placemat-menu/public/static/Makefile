images = cybozu-bird-2.0.aci cybozu-ubuntu-debug-18.04.aci
operation_dir = operation

default: start

cybozu-bird-2.0.aci:
	sudo rkt --insecure-options=image fetch docker://quay.io/cybozu/bird:2.0
	sudo rkt image export quay.io/cybozu/bird:2.0 cybozu-bird-2.0.aci

cybozu-ubuntu-debug-18.04.aci:
	sudo rkt --insecure-options=image fetch docker://quay.io/cybozu/ubuntu-debug:18.04
	sudo rkt image export quay.io/cybozu/ubuntu-debug:18.04 cybozu-ubuntu-debug-18.04.aci

$(operation_dir):
	mkdir -p $(operation_dir)

start: $(images) $(operation_dir)
	@echo "Please run: $(GOPATH)/bin/placemat cluster.yml"

clean:
	sudo rm -rf /var/scratch/placemat

all-clean: clean
	rm -f $(images)
	sudo rm -rf $(HOME)/placemat_data

.PHONY: default start clean all-clean
