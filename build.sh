#!/bin/sh

if [ "$1" = "" ]
then
	echo "Platform is required"
	exit 1
fi

echo $1 | {
  IFS='/' read -r GOOS GOARCH

  export GOOS GOARCH CGO_ENABLED=0

  go build -o ./build/dc4bc_d ./cmd/dc4bc_d/
  go build -o ./build/dc4bc_cli ./cmd/dc4bc_cli/
  go build -o ./build/dc4bc_airgapped ./cmd/airgapped/
  go build -o ./build/dc4bc_dkg_reinitializer ./cmd/dkg_reinitializer/

  sha1sum ./build/dc4bc_* > "./build/checksum.txt"
}