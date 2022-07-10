gocker
===

A minimal psudo-Docker implementation heavily inspired by Liz's Rice [containers-from-scratch](https://github.com/lizrice/containers-from-scratch). This was meant to be a learning exercise after seeing her GOTO 2018 talk titled "Containers From Scratch" and being inspired to further that learning. Much of the base code is from her work and I especially recommend that you check out both the code and the talk!

[![Liz Rice's GOTO 2018 talk titled Containers From Scratch](https://img.youtube.com/vi/8fi7uSYlOdc/0.jpg)](https://www.youtube.com/watch?v=8fi7uSYlOdc)

## Running a Container

> Note: Gocker must be ran as root

To run a command in a container, the command is constructed as follows:

```bash
sudo ./gocker run [IMAGE_OWNER/]IMAGE_NAME[:TAG] [COMMAND]
```

By this, I mean that the image owner, the tag, and the command to run are all optional.

- `IMAGE_OWNER` - Currently Gocker assumes use of Docker Hub. If you are using an official library image, such as the `ubuntu` image, you may omit this and Gocker assumes it's an official library image.
- `TAG` - If omitted, Gocker assumes that it will either use the `latest` tag, or if that doesn't exist, the last tag provided by the Docker Hub API when querying the tags for the image
- `COMMAND` - If omitted, Gocker will defer to using either the `CMD` or `Entrypoint` defined in the containers manifest

For example, since the official Ubuntu image on Docker Hub has a `CMD` defined as `bash`, you could simply run the following and be dropped into a Bash shell:

```bash
sudo ./gocker run ubuntu
```

Alternatively, you can provide a tag and a command to run if you'd prefer:

```bash
sudo ./gocker run ubuntu:22.10 echo hello Gocker

2022/07/10 00:47:04 Running  [echo hello Gocker]  on image  ubuntu:22.10
# Logging omitted for brevity
hello Gocker
```

Or for an exmaple for a non-library image with a tag and a command:

```bash
sudo ./gocker run bitnami/kubectl:1.24 /bin/bash

2022/07/10 00:50:11 Running  [/bin/bash]  on image  bitnami/kubectl:1.24
# Logging omitted for brevity
root@5ZWP5YxLkr4milfpoYNT0erLaDWXDEpS:/ kubectl version

Client Version: version.Info{Major:"1", Minor:"24", GitVersion:"v1.24.2", GitCommit:"f66044f4361b9f1f96f0053dd46cb7dce5e990a8", GitTreeState:"clean", BuildDate:"2022-06-15T14:22:29Z", GoVersion:"go1.18.3", Compiler:"gc", Platform:"linux/amd64"}
Kustomize Version: v4.5.4
```

## Current Limitations

1. Only works with Docker Hub for the container registry
2. There is no networking