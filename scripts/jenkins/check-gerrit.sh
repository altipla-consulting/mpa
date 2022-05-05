#!/bin/bash

set -eux

. /opt/ci-toolset/functions.sh

make lint
make test
