# sync-putio

A little one-way putio syncer thingy. Useful to download (and then **delete
the remote copies of**) files from put.io.

## Quick Start

1. `go build`
2. `cp example.cfg real.cfg && vim real.cfg`
3. `./sync-putio -config real.cfg`

## TODO list

* [x] Multiple sync sources
* [x] User-configurable concurrent downloads
* [x] User-configurable scan interval
* [x] Graceful shutdown everywhere
* [x] File checksum validation
* [x] Rate limiting
* [x] Auto directory creation for bare files
* [ ] Concurrency across multiple sync sources
* [ ] User-configurable deletion semantics
* [ ] Staging / completed folders
