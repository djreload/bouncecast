# BounceCast

BounceCast is a self-hosted livestreaming and chat server forked from Owncast, focused on custom branding, creator tools, and future modular streaming features.

This repository is a maintained fork of [Owncast](https://github.com/owncast/owncast). The first goal is a clean and stable visible rebrand while preserving the working Owncast architecture: livestreaming, RTMP ingest, HLS playback, web chat, admin, Docker, and build behavior.

## About

BounceCast keeps the core self-hosted streaming model from Owncast:

- A Go backend server.
- A React/Next.js frontend and admin interface.
- RTMP ingest for broadcasting software such as OBS.
- HLS playback for viewers.
- Integrated web chat and moderation tools.
- Docker and source builds.

Owncast attribution is intentionally preserved where required and where it helps users understand compatibility with upstream documentation and tooling.

## Getting Started

The quickest path for local development is documented in [DEVELOPER_SETUP_WSL_DEBIAN_13.md](DEVELOPER_SETUP_WSL_DEBIAN_13.md).

For general upstream behavior, the Owncast docs remain useful while BounceCast is still close to upstream:

- [Owncast documentation](https://owncast.online/docs/)
- [Broadcasting with Owncast](https://owncast.online/docs/broadcasting/)
- [Owncast source repository](https://github.com/owncast/owncast)

## Use With Broadcasting Software

BounceCast is compatible with broadcasting software that can stream to an RTMP server. In OBS and similar tools, point your stream output at your BounceCast server and use one of the configured stream keys from the admin panel.

By default:

- Web/admin server: `http://localhost:8080`
- RTMP ingest port: `1935`
- Default stream key on a fresh development install: `abc123`

## Building From Source

BounceCast currently preserves the upstream Owncast Go module path and binary name for compatibility. Do not rename module paths, import paths, database files, API routes, or persisted config keys without a migration plan.

Backend:

```bash
go run main.go
go test ./...
go build -o owncast .
```

Frontend:

```bash
cd web
npm install
npm run dev
npm run build
```

Docker:

```bash
docker build -t bouncecast:dev .
docker run --rm -p 8080:8080 -p 1935:1935 -v bouncecast-data:/app/data bouncecast:dev
```

## Branding Status

Visible project branding is being moved to BounceCast first. Internal Owncast names may remain where they are part of compatibility, generated code, persisted data, tests, build artifacts, or protocol behavior.

See [BOUNCECAST_BRANDING_ASSETS.md](BOUNCECAST_BRANDING_ASSETS.md) for logo/favicon/image work still needed.

## Roadmap

See [BOUNCECAST_ROADMAP.md](BOUNCECAST_ROADMAP.md).

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.

## Attribution

BounceCast is forked from Owncast. Owncast is an open-source self-hosted live video and chat server created by the Owncast project and contributors.
