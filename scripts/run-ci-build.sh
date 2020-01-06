#!/bin/bash

set -xeuo pipefail

root_dir=$(cd $(dirname $BASH_SOURCE)/.. && pwd)

if [[ -n "$TRAVIS_TAG" ]]; then
    echo "Git commit is in master branch and has a commit tag - building and publishing the docker image..."
    ${root_dir}/scripts/build.sh
elif [[ "$TRAVIS_BRANCH" != "master" ]]; then
    echo "Not a release commit. Validating docker container image build only..."
    
    pushd $root_dir
    go build github.com/appbricks/cloud-builder-cli/cmd/cb
    popd
else
    echo "Commit to master branch without tag. Skipping build..."
fi
