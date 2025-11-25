#!/bin/sh
set -e
cd /opt/continuity
#if config file doesn't exist, create it from template
if [ ! -f ./config.yaml ]; then
  continuity-server sample-config
fi
continuity-server