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

To execute the integration test suite, first bring a Tupelo network up by executing the following
commands in the Tupelo repository and waiting for the signer nodes to be ready:
```
make vendor
docker-compose up --build --remove-orphans --force-recreate
```

Then, in the Tupelo Go client repository, execute the following command: `make integration-test`.

## Message Serialization
Before a message gets sent over the wire, it gets serialized to the
[MessagePack](https://msgpack.org/) format. The serialized message gets sent along with a code
describing its type, so that the recipient knows what type of object to deserialize it to.
