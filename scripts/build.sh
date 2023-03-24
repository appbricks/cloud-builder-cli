#!/bin/bash

action=${1:-}
os=${2:-}
arch=${3:-}

set -xeuo pipefail

root_dir=$(cd $(dirname $BASH_SOURCE)/.. && pwd)

run_sudo=''
[[ -z `which sudo` ]] || run_sudo=sudo

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

MYCS_NODE_IMAGE=null

function build() {

  local os=$1
  local arch=$2

  local build_os=$(go env GOOS)

  [[ -n $MYCS_NODE_IMAGE && $MYCS_NODE_IMAGE != null ]] || ( \
    echo "ERROR! Invalid MyCS node image name.";
    exit 1;
  )
  
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
        --env-arg "bastion_image_name=${MYCS_NODE_IMAGE}" \
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
        --env-arg "bastion_image_name=${MYCS_NODE_IMAGE}" \
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
    
    if [[ $build_os == linux ]]; then
      stat -t -c "%Y" cookbook/dist/cookbook.zip > cookbook/dist/cookbook-mod-time
    elif [[ $build_os == darwin ]]; then
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
    # if [[ $build_os == linux && $os == linux ]]; then
    #   if [[ $arch == arm64 ]]; then
    #     # add arm gcc compilers
    #     $run_sudo hwclock --hctosys 
    #     $run_sudo apt update  
    #     $run_sudo apt install -y gcc-aarch64-linux-gnu # for arm8/64 devices (i.e. AWS ARM instances)

    #     GOOS=$os GOARCH=$arch CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 \
    #       go build -ldflags "-s -w $versionFlags" ${root_dir}/cmd/cb
    #   else
    #     GOOS=$os GOARCH=$arch CGO_ENABLED=1 \
    #       go build -ldflags "-s -w $versionFlags" ${root_dir}/cmd/cb
    #   fi
    if [[ $build_os == linux ]]; then
      GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags "-s -w $versionFlags" ${root_dir}/cmd/cb
    else
      GOOS=$os GOARCH=$arch go build -ldflags "-s -w $versionFlags" ${root_dir}/cmd/cb
    fi
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

# List of available MyCS node images. The latest one for
# the environment is passed to the cookbook build to included
# as an environment variable.
mycs_node_images=$(aws ec2 describe-images --output json \
  --region us-east-1 --filters "Name=name,Values=appbricks-bastion*")

if [[ $action == *:dev:* ]]; then

  # Determine MyCS bastion image for dev environment
  dev_images=$(echo "$mycs_node_images" | jq '[.Images[] | select(.Name|test("appbricks-bastion_D.*"))]')
  MYCS_NODE_IMAGE=$(echo "$dev_images" | jq -r 'sort_by(.Name | split("_D.")[1] | split(".") | map(tonumber))[-1] | .Name')

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

  # Determine MyCS bastion image for prod environment
  prod_images=$(echo "$mycs_node_images" | jq '[.Images[] | select(.Name|test("appbricks-bastion_\\d+\\.\\d+\\.\\d+"))'])
  MYCS_NODE_IMAGE=$(echo "$prod_images" | jq -r 'sort_by(.Name | split("_")[1] | split(".") | map(tonumber))[-1] | .Name')

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
