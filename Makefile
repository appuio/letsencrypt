export GOPATH=$(HOME)/gocode

all: ego.go
	go install ./...

$(GOPATH)/bin/ego:
	go get github.com/benbjohnson/ego/cmd/ego

ego.go: $(GOPATH)/bin/ego letsencrypt.ego
	$(GOPATH)/bin/ego --package=letsencrypt .
