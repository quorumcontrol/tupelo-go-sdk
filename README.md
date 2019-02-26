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

### Integration Testing
We launch our integration tests using the
[Tupelo Integration Runner](https://github.com/quorumcontrol/tupelo-integration-runner),
which executes the (integration) test suite against a version of Tupelo. The runner
can be configured to launch the suite for a number of Tupelo versions.

The integration test runner runs the tests inside a Docker container, so this project needs
a Dockerfile from which the runner can build the required image. 

Furthermore, the runner launches the tests container in tandem with a Tupelo network container.
The tests container gets the RPC server host address via an environment variable injected by the
runner, `$TUPELO_RPC_HOST`, so that it can connect during execution.

