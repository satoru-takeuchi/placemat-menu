ignitions = {{range $r := .Racks}}{{$r.Name}}-boot.ign {{$r.Name}}-tor1.ign {{$r.Name}}-tor2.ign \
	{{range $cs := $r.CSList}}{{$r.Name}}-{{$cs.Name}}.ign {{end}}\
	{{range $ss := $r.SSList}}{{$r.Name}}-{{$ss.Name}}.ign {{end}}\
	{{end}}\
	{{range $s := .Spines}}{{$s.Name}}.ign {{end}}
images = cybozu-bird-2.0.aci cybozu-ubuntu-debug-18.04.aci

default: start

$(ignitions): ign.jsonnet ign.libsonnet
	jsonnet -m . ign.jsonnet
	validate $@

cybozu-bird-2.0.aci:
	docker2aci docker://quay.io/cybozu/bird:2.0

cybozu-ubuntu-debug-18.04.aci:
	docker2aci docker://quay.io/cybozu/ubuntu-debug:18.04

start: $(ignitions) $(images)
	@echo "Please run: $(GOPATH)/bin/placemat cluster.yaml"

clean:
	rm -f $(ignitions)
	sudo rm -rf /var/scratch/placemat

all-clean: clean
	rm -f $(images)
	sudo rm -rf $(HOME)/placemat_data

.PHONY: default start clean all-clean
