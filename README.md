# cuardach

*cuardach*: search, or hunt in irish.

privacy-first cli/web search aggregator. ddg and bing support out of the box, with more engines planned. local cache too, and a rotating proxy pool to mask your identity. no telemetry, no accounts, no api keys. just you and the open web.

## Info

what cuardach does:

- rotates useragents constantly
- optionally routes through rotating public proxies or Tor
- stores everything locally.

what cuardach does NOT do:

- send data to any third party (such as Microsoft, Google, OpenAI, etc.)
- log or telemetry of any kind
- phone home or even auto-update.
- require an account or API key


## Install

```bash
git clone https://github.com/cappuch/cuardach
cd cuardach
make build
```

Requires Go 1.24+.

## Quick start

```bash
# generate default config
cuardach config init

# search
cuardach search "golang"

# same query again, instant from cache
cuardach search "golang"

# bypass cache
cuardach search "golang" --no-cache

# use only duckduckgo
cuardach search "linux kernel" -e duckduckgo

# json output
cuardach search "golang" --format json

# start web ui
cuardach serve
# open http://localhost:8080
```

## Commands

```
cuardach search <query>         search
cuardach serve                  start web server (default :8080)
cuardach cache list             list cached queries
cuardach cache stats            cache statistics
cuardach cache purge [--all]    remove expired/all cached data
cuardach cache get <query>      retrieve cached results
cuardach meta search            search local index with filters
cuardach proxy test             fetch and validate public proxies
cuardach proxy list             show proxy list sources
cuardach config init            create default config file
cuardach config show            current config
cuardach version                version info
```

## Config

see [CONFIG.md](CONFIG.md) for full configuration reference.

default config location: `~/.config/cuardach/config.yaml`

## License

MIT
