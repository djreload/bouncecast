# BounceCast Developer Setup: WSL Debian 13

This guide is for developing BounceCast on WSL Debian 13 with Docker.

BounceCast is forked from Owncast. Some internal names, build outputs, Go module paths, database files, and Docker internals still intentionally use `owncast` for compatibility.

## 1. Install System Packages

```bash
sudo apt update
sudo apt install -y git curl ca-certificates build-essential pkg-config ffmpeg make
```

## 2. Install Go

Check the Go version required by the repository:

```bash
grep '^go ' go.mod
```

Install Go from <https://go.dev/dl/> if your Debian package is too old.

Verify:

```bash
go version
```

## 3. Install Node.js

The frontend is a Next.js app. Use Node 20 or newer.

Using NodeSource:

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
node --version
npm --version
```

## 4. Install Docker

Docker Desktop with WSL integration is the easiest option on Windows.

Inside WSL, verify Docker is reachable:

```bash
docker version
docker compose version
```

If Docker is not reachable, open Docker Desktop, enable WSL integration for Debian, then restart the WSL shell.

## 5. Clone The Fork

```bash
cd ~
git clone https://github.com/djreload/bouncecast.git
cd bouncecast
git remote -v
```

Expected remote:

```text
origin  https://github.com/djreload/bouncecast.git (fetch)
origin  https://github.com/djreload/bouncecast.git (push)
```

## 6. Run The Backend

From the repository root:

```bash
go run main.go
```

Open:

```text
http://localhost:8080
http://localhost:8080/admin
```

Fresh development defaults:

```text
Admin password: abc123
RTMP URL: rtmp://localhost:1935/live
Stream key: abc123
```

## 7. Run The Frontend In Development

Use a second WSL terminal and keep the backend running in the first one.

```bash
cd ~/bouncecast/web
npm install
npm run dev
```

Open:

```text
http://localhost:3000
```

The frontend dev server proxies API, HLS, logo, favicon, and related requests to the Go backend on port `8080`.

## 8. Build The Backend

From the repository root:

```bash
go build -o owncast .
```

The output binary is still named `owncast` for compatibility.

Run it:

```bash
./owncast
```

## 9. Build The Frontend

```bash
cd ~/bouncecast/web
npm install
npm run build
```

The frontend uses Next.js static export behavior for production builds.

## 10. Build The Docker Image

From the repository root:

```bash
docker build -t bouncecast:dev .
```

Run it:

```bash
docker run -d \
  --name bouncecast-dev \
  -p 8080:8080 \
  -p 1935:1935 \
  -v bouncecast-data:/app/data \
  bouncecast:dev
```

Open:

```text
http://localhost:8080
http://localhost:8080/admin
```

## 11. View Logs

```bash
docker logs -f bouncecast-dev
```

## 12. Stop And Remove The Container

```bash
docker stop bouncecast-dev
docker rm bouncecast-dev
```

Remove the development data volume only when you want a clean database:

```bash
docker volume rm bouncecast-data
```

## 13. Run Tests

Backend tests:

```bash
go test ./...
```

Backend build:

```bash
go build -o owncast .
```

Frontend tests/build:

```bash
cd web
npm install
npm test
npm run build
```

Make targets:

```bash
make test
make build
```

### Windows Host CGO Setup

When running Go tests from Windows PowerShell instead of WSL, `github.com/mattn/go-sqlite3` needs CGO and a Windows GCC toolchain.

Install MSYS2 and UCRT64 GCC:

```powershell
winget install --id MSYS2.MSYS2 --exact --source winget --accept-package-agreements --accept-source-agreements
C:\msys64\usr\bin\bash.exe -lc "pacman --noconfirm -Syu"
C:\msys64\usr\bin\bash.exe -lc "pacman --noconfirm -S --needed mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-pkgconf make"
```

Add these paths to your user PATH:

```text
C:\msys64\ucrt64\bin
C:\msys64\usr\bin
```

Configure Go to use the MSYS2 compiler:

```powershell
go env -w CGO_ENABLED=1 CC=C:\msys64\ucrt64\bin\gcc.exe CXX=C:\msys64\ucrt64\bin\g++.exe
```

For the current PowerShell session, put MSYS2 first so the newer `make` is used:

```powershell
$machine=[Environment]::GetEnvironmentVariable('Path','Machine')
$user=[Environment]::GetEnvironmentVariable('Path','User')
$env:Path="C:\msys64\ucrt64\bin;C:\msys64\usr\bin;$machine;$user"
```

Verify:

```powershell
go env CGO_ENABLED CC CXX
gcc --version
make --version
go test ./...
```

## 14. Common Troubleshooting

If `go test` fails because `gcc` is missing:

```bash
sudo apt install -y build-essential
```

If video streaming fails, verify ffmpeg is installed:

```bash
ffmpeg -version
```

If `npm install` fails, check Node:

```bash
node --version
npm --version
```

If Docker commands fail inside WSL:

```bash
docker version
```

Then confirm Docker Desktop is running and WSL integration is enabled.

If port `8080`, `1935`, or `3000` is already in use:

```bash
sudo ss -ltnp | grep -E ':8080|:1935|:3000'
```

Stop the conflicting service or choose a different port where the specific tool supports it.

If the admin login does not work on an existing data directory, you may already have a persisted password. Use a fresh data directory for local testing or reset the admin password using the supported Owncast/BounceCast configuration path before changing persisted files directly.
