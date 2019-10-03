#!/usr/bin/env bash

set -eo pipefail

mkdir -p ~/.ssh
echo "$SSH_PRIVATE_KEY" > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa
eval "$(ssh-agent -s)" > /dev/null 2>&1
ssh-add ~/.ssh/id_rsa > /dev/null 2>&1

export GOPATH=${HOME}/go

go mod download

mkdir -p ${GOPATH}/bin

export PATH="${GOPATH}/bin:${PATH}"

# TODO: Run latest release too
git clone git@github.com:quorumcontrol/tupelo.git ~/tupelo
pushd ~/tupelo
make vendor
docker-compose up --build --remove-orphans --force-recreate
popd

echo "Waiting for Tupelo signers to be ready"
sleep 10

make integration-test
