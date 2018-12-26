#!/usr/bin/env sh

set -xeo pipefail

cd /hugo

: ${INTERVAL?"INTERVAL in seconds must be set, e.g. 3600"}
: ${DESTINATION?"DESTINATION must be set, e.g. /my/output/path"}
: ${BACKUP_LOCATION?"BACKUP_LOCATION must be set to the location of the play table json file"}

mkdir $DESTINATION || true

while [ 1 ]; do
	curl -LO $BACKUP_LOCATION

	ruby generate.rb

	hugo

	rm -rf $DESTINATION/*
	mv public/* $DESTINATION/

	echo "Sleeping for $INTERVAL seconds..." && sleep $INTERVAL
done
