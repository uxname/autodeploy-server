.PHONY: build

build:
	rm -f autodeploy.run && go build -ldflags "-w -s" -o autodeploy.run app.go
compress:
	upx -9 autodeploy.run
