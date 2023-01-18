#!/bin/bash

action=${1:-}
os=${2:-}
arch=${3:-}

set -xeuo pipefail

root_dir=$(cd $(dirname $BASH_SOURCE)/.. && pwd)

build_cookbook=../cloud-builder/scripts/build-cookbook.sh
if [[ ! -e $build_cookbook ]]; then
  echo -e "ERROR! Unable to find cookbook build and compilation script."
  exit 1
fi

# install packrv2
[[ -e ${GOPATH}/bin/packr2 ]] || \
  go install github.com/gobuffalo/packr/v2/packr2@latest

build_dir=${root_dir}/.build
if [[ $action == *:clean-all:* ]]; then
  # remove all build artifacts
  # and do a full build
  rm -fr ${build_dir}
fi
release_dir=${build_dir}/releases
mkdir -p ${release_dir}

WINTUN_VER=0.14.1

function build() {

  local os=$1
  local arch=$2
  
  # build cookbook and binary package for given os and arch
  if [[ ! -e ${build_dir}/cookbook/dist/cookbook-${os}_${arch}.zip || $action == *:cookbook:* ]]; then

    local cookbook_repo_path=${COOKBOOK_REPO_PATH:-https://github.com/appbricks/vpn-server/cloud/recipes}
    local cookbook_version=${COOKBOOK_VERSION:-dev}

    # clean packr boxes of cookbook
    pushd ${root_dir}/cmd/cb
    packr2 clean
    popd

    # remove embedded cookbook archive
    rm -fr ${root_dir}/cookbook/dist

    cookbook_desc='This embedded cookbook contains recipes to launch MyCloudSpace space control nodes that manage the network of devices and applications connected to the space network mesh.'

    echo "Building cookbook at $cookbook_repo_path..."
    pushd $root_dir
    if [[ $action == *:clean:* ]]; then
      # clean will remove the cookbook build dist
      $build_cookbook \
        --recipe $cookbook_repo_path \
        --git-branch $cookbook_version \
        --cookbook-name spacenode \
        --cookbook-desc "$cookbook_desc" \
        --cookbook-version $cookbook_version \
        --os-name $os \
        --os-arch $arch \
        --clean --verbose
    else
      $build_cookbook \
        --recipe $cookbook_repo_path \
        --git-branch $cookbook_version \
        --cookbook-name spacenode \
        --cookbook-desc "$cookbook_desc" \
        --cookbook-version $cookbook_version \
        --os-name $os \
        --os-arch $arch \
        --verbose
    fi
    popd

  else
    set +e
    # ensure embedded cookbook is the correct one for the given os and arch
    diff ${build_dir}/cookbook/dist/cookbook-${os}_${arch}.zip cookbook/dist/cookbook.zip 2>&1 >/dev/null
    if [[ $? -ne 0 ]]; then
      set -e
      # clean packr boxes of cookbook
      pushd ${root_dir}/cmd/cb
      packr2 clean
      popd

      cp ${build_dir}/cookbook/dist/cookbook-${os}_${arch}.zip cookbook/dist/cookbook.zip
    else
      set -e
    fi
    
    local current_os=$(go env GOOS)
    if [[ $current_os == linux ]]; then
      stat -t -c "%Y" cookbook/dist/cookbook.zip > cookbook/dist/cookbook-mod-time
    elif [[ $current_os == darwin ]]; then
      stat -t "%s" -f "%Sm" cookbook/dist/cookbook.zip > cookbook/dist/cookbook-mod-time
    else
      echo -e "\nERROR! Unable to get the modification timestamp of 'cookbook/dist/cookbook.zip'.\n"
      exit 1
    fi

  fi
  if [[ ! -e ${root_dir}/cmd/cb/packrd ]]; then
    # create packr boxes of cookbook
    pushd ${root_dir}/cmd/cb
    packr2 -v
    popd
  fi

  # build and package release binary
  mkdir -p ${release_dir}/${os}_${arch}
  pushd ${release_dir}/${os}_${arch}

  versionFlags="-X \"github.com/appbricks/cloud-builder-cli/config.Version=$build_version\" -X \"github.com/appbricks/cloud-builder-cli/config.BuildTimestamp=$build_timestamp\""
  
  if [[ $action == *:dev:* ]]; then
    GOOS=$os GOARCH=$arch go build -ldflags "$versionFlags" ${root_dir}/cmd/cb
  else
    GOOS=$os GOARCH=$arch go build -ldflags "-s -w $versionFlags" ${root_dir}/cmd/cb
  fi
  if [[ $os == windows ]]; then
    curl -OL https://www.wintun.net/builds/wintun-${WINTUN_VER}.zip
    unzip wintun-${WINTUN_VER}.zip
    rm wintun-${WINTUN_VER}.zip
    cp wintun/bin/${arch}/wintun.dll .
    rm -fr wintun
  fi
  zip -r ${release_dir}/cb_${os}_${arch}.zip .
  popd
}

if [[ $action == *:dev:* ]]; then
  # set version
  build_version=dev
  build_timestamp=$(date +'%B %d, %Y at %H:%M %Z')

  # build binary for a dev environment
  rm -f $GOPATH/bin/cb

  os=$(go env GOOS)
  arch=$(go env GOARCH)
  build "$os" "$arch"
  ln -s ${release_dir}/${os}_${arch}/cb $GOPATH/bin/cb

elif [[ $action == *:release:* ]]; then
  # set version
  tag=${GITHUB_REF/refs\/tags\//}
  build_version=${tag:-0.0.0}
  build_timestamp=$(date +'%B %d, %Y at %H:%M %Z')

  # build release binaries for all supported architectures
  if [[ -n $os && -n $arch ]]; then
    build "$os" "$arch"
  else
    build "linux" "amd64"
    build "linux" "arm64"
    build "darwin" "amd64"
    build "darwin" "arm64"
    build "windows" "amd64"
  fi
else
  echo "ERROR! Invald build action: $action"
  exit 1
fi
