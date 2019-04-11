#!/bin/bash
set -eo pipefail

mkdir -p ~/.ssh
# echo "$SSH_KEY" > ~/.ssh/id_rsa
# chmod 600 ~/.ssh/id_rsa
# eval `ssh-agent`
# ssh-add ~/.ssh/id_rsa
git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts

make lint

if [[ "${CI}" == "true" ]]; then
  make ci-test
else
  make test
fi
