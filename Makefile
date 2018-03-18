all:
	$(MAKE) deps
	$(MAKE) unitymetrics

deps:
	go get -u github.com/equelin/gounity
	go get -u github.com/sirupsen/logrus

unitymetrics:
	env GOOS=linux GOARCH=amd64 go build -v github.com/equelin/unitymetrics
	env GOOS=windows GOARCH=amd64 go build -v github.com/equelin/unitymetrics

go-install:
	go install 

install: unitymetrics
	cp ./unitymetrics /usr/local/bin

