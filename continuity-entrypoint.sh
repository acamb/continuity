#!/bin/sh
set -e
cd /opt/continuity
#if config file doesn't exist, create it from template
if [ ! -f ./config.yaml ]; then
  echo "No config found, creating default config..."
  /usr/bin/continuity-server -sample-config
  echo "Created default config:"
  cat ./config.yaml
fi
/usr/bin/continuity-server