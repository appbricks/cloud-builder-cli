#!/bin/bash

set -xeuo pipefail

root_dir=$(cd $(dirname $BASH_SOURCE)/.. && pwd)
action=${1:-}

if [[ -z $action || $action == *:cookbook:* ]]; then

  build_cookbook=../cloud-builder/scripts/build-cookbook.sh
  if [[ ! -e $build_cookbook ]]; then
    echo -e "ERROR! Unable to find cookbook build and compilation script."
    exit 1
  fi

  cookbook_repo_path=${COOKBOOK_REPO_PATH:-https://github.com/appbricks/vpn-server/cloud/recipes}
  cookbook_version=${COOKBOOK_VERSION:-dev}

  echo "Building cookbook at $cookbook_repo_path..."
  pushd $root_dir
  if [[ -z $action || $action == *:clean:* ]]; then
    $build_cookbook -r $cookbook_repo_path -b $cookbook_version -c
  else
    $build_cookbook -r $cookbook_repo_path -b $cookbook_version
  fi
  popd

  # clean packr boxes of cookbook
  pushd ${root_dir}/cmd/cb
  packr2 clean
  popd
fi
if [[ ! -e ${root_dir}/cmd/cb/packrd ]]; then
  # create packr boxes of cookbook
  pushd ${root_dir}/cmd/cb
  packr2 -v
  popd
fi

# build and package binary for given os and arch
function build_binary() {

  local release_dir=$1
  local os=$2
  local arch=$3

  mkdir -p ${release_dir}/${os}_${arch}
  GOOS=$os GOARCH=$arch go build -o ${release_dir}/${os}_${arch}/cb ${root_dir}/cmd/cb
  tar -cvzf ${release_dir}/cb_${os}_${arch}.tgz -C ${release_dir}/${os}_${arch}/ .
}

release_dir=${root_dir}/build/releases
if [[ -z $action || $action == *:clean:* ]]; then
  rm -fr $release_dir
fi
if [[ $action == *:dev:* ]]; then
  # build binary for a dev environment
  rm -f $GOPATH/bin/cb
  if [[ $(uname) == Darwin ]]; then
    build_binary "$release_dir" "darwin" "amd64"
    ln -s ${release_dir}/darwin_amd64/cb $GOPATH/bin/cb
  else
    build_binary "$release_dir" "linux" "amd64"
    ln -s ${release_dir}/linux_amd64/cb $GOPATH/bin/cb
  fi
elif [[ -z $action || $action == *:release:* ]]; then
  # build release binary
  build_binary "$release_dir" "darwin" "amd64"
  build_binary "$release_dir" "linux" "amd64"
  build_binary "$release_dir" "windows" "amd64"
fi
