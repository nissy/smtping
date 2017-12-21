SMTPING_VERSION := 0.1.0
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

ifeq ($(GOOS),windows)
dist/smtping-$(SMTPING_VERSION)_$(GOOS)_$(GOARCH).zip:
	go build -ldflags="-X main.Version=$(SMTPING_VERSION)" -o dist/smtping-$(SMTPING_VERSION)/smtping.exe cmd/main.go
	zip -j dist/smtping-$(SMTPING_VERSION)_$(GOOS)_$(GOARCH).zip dist/smtping-$(SMTPING_VERSION)/smtping.exe
else
dist/smtping-$(SMTPING_VERSION)_$(GOOS)_$(GOARCH).tar.gz:
	go build -ldflags="-X main.Version=$(SMTPING_VERSION)" -o dist/smtping-$(SMTPING_VERSION)/smtping cmd/main.go
	tar cfz dist/smtping-$(SMTPING_VERSION)_$(GOOS)_$(GOARCH).tar.gz -C dist/smtping-$(SMTPING_VERSION) smtping
endif

.PHONY: clean
clean:
	rm -rf dist
