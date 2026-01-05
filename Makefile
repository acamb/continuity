.PHONY: all server client client-windows clean deb client-deb docker-client-deb rpm client-rpm docker-server-rpm docker-client-rpm release

BINDIR := bin
PKGDIR := pkg
VERSION := $(shell cat version)
DEBNAME := $(BINDIR)/continuity-server_$(VERSION)_amd64.deb
DEBNAME_CLIENT := $(BINDIR)/continuity_$(VERSION)_amd64.deb
RPMNAME := $(BINDIR)/continuity-server-$(VERSION)-1.x86_64.rpm
RPMNAME_CLIENT := $(BINDIR)/continuity-$(VERSION)-1.x86_64.rpm

all: server client

server:
	mkdir -p $(BINDIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'continuity/server/version.Version=$(VERSION)'" -o $(BINDIR)/continuity-server_$(VERSION) ./server/cmd

server-static:
	mkdir -p $(BINDIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X 'continuity/server/version.Version=$(VERSION)'" -o $(BINDIR)/continuity-server-static_$(VERSION) ./server/cmd

client:
	mkdir -p $(BINDIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'continuity/client/version.Version=$(VERSION)'" -o $(BINDIR)/continuity_$(VERSION) ./client/cmd

client-windows:
	mkdir -p $(BINDIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "-X 'continuity/client/version.Version=$(VERSION)'" -o $(BINDIR)/continuity_$(VERSION).exe ./client/cmd

clean:
	rm -rf $(BINDIR) $(PKGDIR) $(DEBNAME) $(DEBNAME_CLIENT) $(RPMNAME) $(RPMNAME_CLIENT)

deb:
	rm -rf $(PKGDIR)
	mkdir -p $(PKGDIR)/DEBIAN
	mkdir -p $(PKGDIR)/usr/bin
	mkdir -p $(PKGDIR)/lib/systemd/system
	mkdir -p $(PKGDIR)/opt/continuity
	cp $(BINDIR)/continuity-server_$(VERSION) $(PKGDIR)/usr/bin/continuity-server
	cp continuity-server.service $(PKGDIR)/lib/systemd/system/
	cp config-default.yml $(PKGDIR)/opt/continuity/config.yaml
	echo "Package: continuity-server\nVersion: $(VERSION)\nSection: base\nPriority: optional\nArchitecture: amd64\nMaintainer: Your Name <you@example.com>\nDescription: continuity server" > $(PKGDIR)/DEBIAN/control
	echo "#!/bin/sh\n\
    id -u continuity-server >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin continuity-server\n\
    chown -R continuity-server:continuity-server /opt/continuity\n\
    systemctl daemon-reload\n\
    systemctl enable continuity-server.service\n\
    systemctl start continuity-server.service\n\
    " > $(PKGDIR)/DEBIAN/postinst
	chmod 755 $(PKGDIR)/DEBIAN/postinst
	dpkg-deb --build $(PKGDIR) $(DEBNAME)

rpm:
	rm -rf $(PKGDIR)-rpm
	mkdir -p $(PKGDIR)-rpm/BUILDROOT/continuity-server-$(VERSION)-1.x86_64/usr/bin
	mkdir -p $(PKGDIR)-rpm/BUILDROOT/continuity-server-$(VERSION)-1.x86_64/lib/systemd/system
	mkdir -p $(PKGDIR)-rpm/BUILDROOT/continuity-server-$(VERSION)-1.x86_64/opt/continuity
	mkdir -p $(PKGDIR)-rpm/SPECS
	cp $(BINDIR)/continuity-server_$(VERSION) $(PKGDIR)-rpm/BUILDROOT/continuity-server-$(VERSION)-1.x86_64/usr/bin/continuity-server
	cp continuity-server.service $(PKGDIR)-rpm/BUILDROOT/continuity-server-$(VERSION)-1.x86_64/lib/systemd/system/
	cp config-default.yml $(PKGDIR)-rpm/BUILDROOT/continuity-server-$(VERSION)-1.x86_64/opt/continuity/config.yaml
	echo "Name: continuity-server" > $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "Version: $(VERSION)" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "Release: 1" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "Summary: continuity server" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "License: MIT" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "Group: System Environment/Daemons" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "BuildArch: x86_64" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "Requires: systemd" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "%description" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "continuity server for application deployment" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "%post" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "id -u continuity-server >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin continuity-server" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "chown -R continuity-server:continuity-server /opt/continuity" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "systemctl daemon-reload" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "systemctl enable continuity-server.service" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "systemctl start continuity-server.service" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "%files" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "/usr/bin/continuity-server" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "/lib/systemd/system/continuity-server.service" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	echo "/opt/continuity/config.yaml" >> $(PKGDIR)-rpm/SPECS/continuity-server.spec
	cd $(PKGDIR)-rpm && rpmbuild --define "_topdir $(PWD)/$(PKGDIR)-rpm" --define "_rpmdir $(PWD)/$(BINDIR)" -bb SPECS/continuity-server.spec

docker-server-deb-container:
	docker run --rm -v $(PWD):/workspace -w /workspace debian:bookworm bash -c "\
	apt-get update && \
	apt-get install -y ca-certificates make dpkg-dev && \
	make deb && \
	chown -R $(shell id -u):$(shell id -g) /workspace \
	"

docker-server-deb: server
	make docker-server-deb-container

client-deb:
	rm -rf $(PKGDIR)-client
	mkdir -p $(PKGDIR)-client/DEBIAN
	mkdir -p $(PKGDIR)-client/usr/bin
	cp $(BINDIR)/continuity_$(VERSION) $(PKGDIR)-client/usr/bin/continuity
	echo "Package: continuity\nVersion: $(VERSION)\nSection: base\nPriority: optional\nArchitecture: amd64\nMaintainer: Your Name <you@example.com>\nDescription: continuity client CLI tool" > $(PKGDIR)-client/DEBIAN/control
	dpkg-deb --build $(PKGDIR)-client $(DEBNAME_CLIENT)

client-rpm:
	rm -rf $(PKGDIR)-client-rpm
	mkdir -p $(PKGDIR)-client-rpm/BUILDROOT/continuity-$(VERSION)-1.x86_64/usr/bin
	mkdir -p $(PKGDIR)-client-rpm/SPECS
	cp $(BINDIR)/continuity_$(VERSION) $(PKGDIR)-client-rpm/BUILDROOT/continuity-$(VERSION)-1.x86_64/usr/bin/continuity
	echo "Name: continuity" > $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "Version: $(VERSION)" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "Release: 1" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "Summary: continuity client CLI tool" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "License: MIT" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "Group: Applications/System" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "BuildArch: x86_64" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "%description" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "continuity client CLI tool for application deployment" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "%files" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	echo "/usr/bin/continuity" >> $(PKGDIR)-client-rpm/SPECS/continuity.spec
	cd $(PKGDIR)-client-rpm && rpmbuild --define "_topdir $(PWD)/$(PKGDIR)-client-rpm" --define "_rpmdir $(PWD)/$(BINDIR)" -bb SPECS/continuity.spec

docker-client-deb-container:
	docker run --rm -v $(PWD):/workspace -w /workspace debian:bookworm bash -c "\
	apt-get update && \
	apt-get install -y ca-certificates make dpkg-dev && \
	make client-deb && \
	chown -R $(shell id -u):$(shell id -g) /workspace \
	"

docker-client-deb: client
	make docker-client-deb-container

docker-server-rpm-container:
	docker run --rm -v $(PWD):/workspace -w /workspace rockylinux:9 bash -c "\
	yum update -y && \
	yum install -y ca-certificates make rpm-build && \
	make rpm && \
	chown -R $(shell id -u):$(shell id -g) /workspace \
	"

docker-server-rpm: server
	make docker-server-rpm-container

docker-client-rpm-container:
	docker run --rm -v $(PWD):/workspace -w /workspace rockylinux:9 bash -c "\
	yum update -y && \
	yum install -y ca-certificates make rpm-build && \
	make client-rpm && \
	chown -R $(shell id -u):$(shell id -g) /workspace \
	"

docker-client-rpm: client
	make docker-client-rpm-container

docker-release: clean server-static
	docker build -t continuity-server -f Dockerfile .
	docker tag continuity-server acamb23/continuity-server:$(VERSION)
	docker tag continuity-server acamb23/continuity-server

docker-publish: docker-release
	docker push acamb23/continuity-server:$(VERSION)
	docker push acamb23/continuity-server:latest

release: server server-static client client-windows docker-server-deb docker-client-deb docker-server-rpm docker-client-rpm

check:
	cd server/api && go test -v ./... && cd ..
	cd server/conf && go test -v ./... && cd ../..
