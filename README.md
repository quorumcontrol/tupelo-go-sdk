[![CircleCI](https://circleci.com/gh/quorumcontrol/tupelo-go-sdk.svg?style=svg&circle-token=7e9e6a1638c33dcb8899cc9a2aed9936cba60aaa)](https://circleci.com/gh/quorumcontrol/tupelo-go-sdk)

# Tupelo Go SDK
This is the SDK for developing Go applications using [Tupelo](https://quorumcontrol.com).

## Building
Before building the project, you should have the following installed:

* Go >= v1.12
* Docker
* GNU Make

### Docker Image
In order to build a Docker image, you first have to prepare the vendor tree, so that it contains
all the dependencies so we can just copy them into the Docker build process. Our Makefile will
take care of this for you though, just invoke `make` like this:

```
make docker-image
```

## Testing

In order to run the regular test suite (i.e. non-integration tests), execute the following
command: `make test`.

### Integration Testing
We run our integration tests in a Docker container, against a network of Tupelo signers
(and a bootstrap node) launched via `docker-compose`. The tests must run in a container in order
to connect to the network created by `docker-compose`.

You can run integration tests against Tupelo master (the default), the latest release of Tupelo,
or against your own local working copy of Tupelo (which should be in a sibling directory named
`tupelo`; override the `TUPELO_DIR` variable if it is somewhere else).

To run the tests against Tupelo...
    
    * master: `make integration-test`
    * latest release: `make integration-test TUPELO=latest`
    * local working copy: `make integration-test TUPELO=local`

## Message Serialization
Before a message gets sent over the wire, it gets serialized to the
[MessagePack](https://msgpack.org/) format. The serialized message gets sent along with a code
describing its type, so that the recipient knows what type of object to deserialize it to.
