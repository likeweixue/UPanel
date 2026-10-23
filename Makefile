.PHONY: build build-cli build-all clean install web release test vet

# 版本号（默认读取 VERSION 文件）
VERSION?=$(shell cat VERSION 2>/dev/null || echo v0.1.7)

# Go 环境
export CGO_ENABLED=0

# 编译主程序到 bin/
build:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o bin/upanel cmd/upanel/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o bin/up cmd/upanel/cli/main.go

# 编译所有平台
build-all:
	mkdir -p release
	# Linux amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/linux-amd64/upanel cmd/upanel/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/linux-amd64/up cmd/upanel/cli/main.go
	# Linux arm64
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/linux-arm64/upanel cmd/upanel/main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/linux-arm64/up cmd/upanel/cli/main.go
	# macOS amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/darwin-amd64/upanel cmd/upanel/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/darwin-amd64/up cmd/upanel/cli/main.go
	# macOS arm64 (M 系列)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/darwin-arm64/upanel cmd/upanel/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=${VERSION}" -o release/darwin-arm64/up cmd/upanel/cli/main.go

# 构建前端
web:
	cd web && npm install --no-audit --no-fund && npm run build

# 生成离线安装发布包（前端 + 后端 + 安装脚本）
release: web
	bash scripts/build-release.sh ${VERSION}

# 仅打包（前端已构建过时使用）
release-fast:
	bash scripts/build-release.sh ${VERSION}

# 静态检查
vet:
	go vet ./...

# 静态检查 + 单元测试
test: vet
	go build ./...
	go test ./...

# 清理
clean:
	rm -rf bin/ build/
	rm -rf web/dist

# 安装到本地
install: build
	cp bin/upanel /usr/local/bin/
	cp bin/up /usr/local/bin/
