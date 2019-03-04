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
docker-compose -f docker-compose-signers.yml up --build --remove-orphans --force-recreate
```

Then, in the Tupelo Go client repository, execute the following commands:
```
make docker-image
docker run -e TUPELO_BOOTSTRAP_NODES=/ip4/172.16.238.10/tcp/34001/ipfs/ \
QmW2hgZqe6UcQ6kTaF8kS6CA3RDo7wbCvnGctCetbSt85n --net tupelo_default \
quorumcontrol/tupelo-go-client go test -tags=integration -timeout=2m ./...
```
