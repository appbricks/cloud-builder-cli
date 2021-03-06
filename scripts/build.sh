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
[[ -e ${GOPATH}/bin/packr2 ]] || \
  go get -u github.com/gobuffalo/packr/v2/packr2

build_dir=${root_dir}/build
if [[ $action == *:clean-all:* ]]; then
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
  if [[ ! -e ${build_dir}/cookbook/dist/cookbook-${os}_${arch}.zip || $action == *:cookbook:* ]]; then

    local cookbook_repo_path=${COOKBOOK_REPO_PATH:-https://github.com/appbricks/vpn-server/cloud/recipes}
    local cookbook_version=${COOKBOOK_VERSION:-refactor}

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
  GOOS=$os GOARCH=$arch go build ${root_dir}/cmd/cb
  zip -r ${release_dir}/cb_${os}_${arch}.zip .
  popd
}

if [[ $action == *:dev:* ]]; then
  # build binary for a dev environment
  rm -f $GOPATH/bin/cb

  os=$(go env GOOS)
  arch=$(go env GOARCH)
  build "$os" "$arch"
  ln -s ${release_dir}/${os}_${arch}/cb $GOPATH/bin/cb

else
  # set version
  build_version=${TRAVIS_TAG:-0.0.2}
  build_timestamp=$(date +'%B %d, %Y at %H:%M %Z')

  if [[ `go env GOOS` == darwin ]]; then
    sed -i '' \
      "s|^const VERSION = \`.*\`$|const VERSION = \`$build_version\`|" \
      ${root_dir}/cmd/version.go
    sed -i '' \
      "s|^const BUILD_TIMESTAMP = \`.*\`$|const BUILD_TIMESTAMP = \`$build_timestamp\`|" \
      ${root_dir}/cmd/version.go
  else
    sed -i \
      "s|^const VERSION = \`.*\`$|const VERSION = \`$build_version\`|" \
      ${root_dir}/cmd/version.go
    sed -i \
      "s|^const BUILD_TIMESTAMP = \`.*\`$|const BUILD_TIMESTAMP = \`$build_timestamp\`|" \
      ${root_dir}/cmd/version.go
  fi

  # build release binaries for all supported architectures
  build "darwin" "amd64"
  build "linux" "amd64"
  build "windows" "amd64"
fi
