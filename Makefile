# This is how we want to name the binary output
NAME=/media/cobolbaby/data/ubuntu/opt/workspace/git/dc-agent-release/v2.0/dcagent

# These are the values we want to pass for Version and BuildTime
GIT_COMMIT=`git rev-parse --short HEAD`
BUILD_TIME=`date +%FT%T%z`
GO_VERSION=`go version`

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-ldflags "-w -X main.GIT_COMMIT=${GIT_COMMIT} -X 'main.BUILD_TIME=${BUILD_TIME}' -X 'main.GO_VERSION=${GO_VERSION}'"

build:
	# go clean -modcache
	GO111MODULE=on go mod vendor
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build ${LDFLAGS} -o $(NAME)_2.0.1_windows_x86.exe main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o $(NAME)_2.0.1_windows_amd64.exe main.go
	/opt/programs/upx/upx -f -9 $(NAME)_2.0.1_windows_x86.exe
	/opt/programs/upx/upx -f -9 $(NAME)_2.0.1_windows_amd64.exe

install:
	make build
	# mv -v $(NAME) $(GOPATH)/bin/$(NAME)
