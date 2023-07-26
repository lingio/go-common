#!/usr/bin/env bash

set -euo pipefail

PROJECT=$1
INSTANCE=$2
DATABASE=$3

echo "* Setting up local spanner emulator using schema and data from:"
echo ""
echo "    projects/$PROJECT/instances/$INSTANCE/databases/$DATABASE"
echo ""

SPANNER_EMULATOR_HOST=$(gcloud emulators spanner env-init | cut -d= -f2)
SPANNER_EMULATOR_ADDR=$(cut -d: -f1 <<< $SPANNER_EMULATOR_HOST)
SPANNER_EMULATOR_PORT=$(cut -d: -f2 <<< $SPANNER_EMULATOR_HOST)

if nc -zv $SPANNER_EMULATOR_ADDR $SPANNER_EMULATOR_PORT 2>/dev/null; then
	echo "* Spanner emulator is running"
else
	echo "* Waiting for spanner emulator... "
	echo ""
	echo "  To start, run cmd in another terminal:"
	echo ""
	echo "  gcloud emulators spanner start"
	echo ""
	while true; do
		sleep 3
		if nc -zv $SPANNER_EMULATOR_ADDR $SPANNER_EMULATOR_PORT 2>/dev/null; then
			echo "* Spanner emulator is running"
			break
		else
			echo "* Waiting for spanner emulator... "
		fi
	done
fi

echo "* Installing spanner-cli and spanner-tools ..."
go install github.com/cloudspannerecosystem/spanner-cli@latest
go install github.com/lingio/go-common/script/spanner-tools@latest

echo "* Activating gcloud emulator config"
gcloud config configurations activate emulator

echo "* Checking for spanner instance..."
if gcloud spanner instances list |  grep -q test-instance; then
	echo "* Found existing spanner instance"
else
	echo "* No instance found, creating ..."
	gcloud spanner instances create test-instance \
		--config=emulator-config \
		--description="Test Instance" \
		--nodes=1
fi

echo "* Exporting schema from $PROJECT ..."
mkdir -p sql
wrench \
	--project $PROJECT \
	--instance $INSTANCE \
	--database $DATABASE \
	--directory sql \
	load

echo "* Exporting data from $PROJECT ..."
spanner-tools \
	-p $PROJECT \
	-i $INSTANCE \
	-d $DATABASE \
	--limit 500 \
	row list > stage-$DATABASE.jsonl

ls -lh stage-$DATABASE.jsonl


echo "* Sourcing spanner emulator env"
source <(gcloud emulators spanner env-init);

echo "* Resetting database ..."
wrench \
	--project lingio-test \
	--instance test-instance \
	--database $DATABASE \
	--directory sql \
	reset

echo "* Importing data from $PROJECT ..."
spanner-tools \
	-p lingio-test \
	-i test-instance \
	-d $DATABASE \
	row insert < stage-$DATABASE.jsonl


echo "* Emulator instance is now ready!"
echo ""
echo "  Table rows:"
spanner-cli \
	-p lingio-test \
	-i test-instance \
	-d $DATABASE \
	-e="SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE';" | \
	grep -v "TABLE_NAME" | \
	xargs -I {} spanner-cli \
		-p lingio-test \
		-i test-instance \
		-d $DATABASE \
		-e="SELECT '{}' as name, count(*) as x FROM {}" | \
		grep -v name

if -f sql/migrations; then
	echo "* Running sql migrations ..."
	wrench \
		--project lingio-test \
		--instance test-instance \
		--database $DATABASE \
		--directory sql \
		migrate up
else
	echo "* sql/migrations dir not found, skipping migrations"
fi

echo ""
echo " Spanner database path:"
echo ""
echo "	projects/lingio-test/instances/test-instance/$DATABASE"
echo ""
echo "* Run service with SPANNER_EMULATOR_HOST=$SPANNER_EMULATOR_HOST env. to dial emulator,"
echo "  and remember to use the above database path."
echo ""
