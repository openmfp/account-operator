#!/usr/bin/env bash

DEFAULT_RELATIVE_BIN_PATH="bin/kcp"
RELATIVE_BIN_PATH=${1:-$DEFAULT_RELATIVE_BIN_PATH}

# test if kcp binary exists, if it does exit
if [ -f $RELATIVE_BIN_PATH ]; then
    echo "kcp binary already exists, skipping build"
    exit 0
fi

# This script should download the kcp repo to a directory and build it
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Create tmp directory
rm -rf ${SCRIPT_DIR}/tmp
mkdir -p ${SCRIPT_DIR}/tmp/kcp
trap "rm -rf ${SCRIPT_DIR}/tmp" EXIT
git clone https://github.com/kcp-dev/kcp.git ${SCRIPT_DIR}/tmp/kcp

# Build kcp
cd ${SCRIPT_DIR}/tmp/kcp
IGNORE_GO_VERSION=1  make build

cp ${SCRIPT_DIR}/tmp/kcp/bin/kcp ${SCRIPT_DIR}/../$RELATIVE_BIN_PATH