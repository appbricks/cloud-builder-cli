#!/bin/bash

set -xeuo pipefail

root_dir=$(cd $(dirname $BASH_SOURCE)/.. && pwd)
action=${1:-}

build_cookbook=../cloud-builder/scripts/build-cookbook.sh
if [[ ! -e $build_cookbook ]]; then
  echo -e "ERROR! Unable to find cookbook build and compilation script."
  exit 1
fi

# install packrv2
go get -u github.com/gobuffalo/packr/v2/packr2

build_dir=${root_dir}/build
if [[ $action == *:clean_all:* ]]; then
  # remove all build artifacts
  # and do a full build
  rm -fr ${build_dir}
fi
release_dir=${build_dir}/releases
mkdir -p ${release_dir}

function build() {

  local os=$1
  local arch=$2
  
  # build cookbook and binary package for given os and arch
  if [[ ! -e ${build_dir}/cookbook/dist/${os}_${arch} || $action == *:cookbook:* ]]; then

    local cookbook_repo_path=${COOKBOOK_REPO_PATH:-https://github.com/appbricks/vpn-server/cloud/recipes}
    local cookbook_version=${COOKBOOK_VERSION:-dev}

    # clean packr boxes of cookbook
    pushd ${root_dir}/cmd/cb
    packr2 clean
    popd

    # remove embedded cookbook archive
    rm -fr ${root_dir}/cookbook/dist

    echo "Building cookbook at $cookbook_repo_path..."
    pushd $root_dir
    if [[ $action == *:clean:* ]]; then
      # clean will remove the cookbook build dist
      $build_cookbook -r $cookbook_repo_path -b $cookbook_version -o $os -c -v
    else
      $build_cookbook -r $cookbook_repo_path -b $cookbook_version -o $os -v
    fi
    popd
  fi
  if [[ ! -e ${root_dir}/cmd/cb/packrd ]]; then
    # create packr boxes of cookbook
    pushd ${root_dir}/cmd/cb
    packr2 -v
    popd
  fi

  # build and package release binary
  mkdir -p ${release_dir}/${os}_${arch}
  GOOS=$os GOARCH=$arch go build -o ${release_dir}/${os}_${arch}/cb ${root_dir}/cmd/cb
  tar -cvzf ${release_dir}/cb_${os}_${arch}.tgz -C ${release_dir}/${os}_${arch}/ .
}

if [[ $action == *:dev:* ]]; then
  # build binary for a dev environment
  rm -f $GOPATH/bin/cb

  os=$(go env GOOS)
  arch=$(go env GOARCH)
  build "$os" "$arch"
  ln -s ${release_dir}/${os}_${arch}/cb $GOPATH/bin/cb

elif [[ -z $action || $action == *:release:* ]]; then
  # build release binaries for all supported architectures
  build "darwin" "amd64"
  build "linux" "amd64"
  build "windows" "amd64"
fi
