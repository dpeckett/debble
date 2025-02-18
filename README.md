# immutos

A declarative, BuildKit powered, Debian base system builder. 

Inspired by [apko](https://github.com/chainguard-dev/apko), [multistrap](https://wiki.debian.org/Multistrap), 
[debootstrap](https://wiki.debian.org/Debootstrap), and [cloud-init](https://cloudinit.readthedocs.io/en/latest/).

## Features

* Declarative - specify your base system in a YAML file.
* Reproducible - run the same command, get the same image.
* Secure - no need to trust a third party base image.
* Fast - uses [BuildKit](https://docs.docker.com/build/buildkit/) for caching and parallelism.
* Portable - build images on any platform that supports Docker.

## Installation

### From APT

Add the [immutos apt repository](https://github.com/immutos/apt?tab=readme-ov-file#usage) to your system.

Then install immutos:

*Currently packages are only published for Debian 12 (Bookworm).*

```shell
sudo apt update
sudo apt install -y immutos
```

### GitHub Releases

Download statically linked binaries from the GitHub releases page: 

[Latest Release](https://github.com/immutos/immutos/releases/latest)

## Usage

### Prerequisites

* Docker

### Building a Image

To create a minimal Debian image:

```shell
immutos build -f examples/bookworm-ultraslim.yaml
```

The resulting OCI archive will be saved to `debian-image.tar`.

### Running the Image

You will need a recent release of the [Skopeo](https://github.com/containers/skopeo) 
(eg. v1.15.1) to copy the image into your Docker daemon cache as Docker does not 
have native support for loading OCI images.

```shell
skopeo copy oci-archive:debian-image.tar docker-daemon:immutos/debian:bookworm-ultraslim
```

You can then run the image with:

```shell
docker run --rm -it immutos/debian:bookworm-ultraslim sh
```

### Using a Prebuilt Image

For convenience the immutos build pipeline publishes a bookworm-ultraslim image.
This image is intended for experimentation purposes only. You should build your
own base images using the `immutos build` command.

```shell
docker run --rm -it registry.dpeckett.dev/immutos/immutos/debian:bookworm-ultraslim
```

## Telemetry

By default immutos gathers anonymous crash and usage statistics. This anonymized
data is processed on our servers within the EU and is not shared with third
parties. You can opt out of telemetry by setting the `DO_NOT_TRACK=1`
environment variable.

## Limitations

* [Debian Bookworm](https://www.debian.org/releases/bookworm/) and newer.