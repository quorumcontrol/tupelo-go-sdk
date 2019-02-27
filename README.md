# Tupelo Go Client
This is the SDK for developing Go applications using [Tupelo](https://quorumcontrol.com).

## Building
Before building the project, you should have the following installed:

* Go >= v1.11
* Docker
* GNU Make

### Docker Image
In order to build a Docker image, you first have to prepare the vendor tree, so that it contains
all the dependencies so we can just copy them into the Docker build process. Our Makefile will
take care of this for you though, just invoke `make` like this:

```
make docker-image
```
