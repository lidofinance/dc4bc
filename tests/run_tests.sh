#!/bin/bash

echo "-=Stop old docker-compose=-"
docker-compose down -v
#docker stop dc4bc-kafka && docker rm dc4bc-kafka
#docker stop dc4bc-zookeeper && docker rm dc4bc-zookeeper

echo "-=Start new docker-compose=-"
docker-compose up -d --build dc4bc-kafka dc4bc-zookeeper

# Required for Kafka to get ready.
sleep 30

echo "-=Start tests=-"
cd ../

# shellcheck disable=SC2046
go test $(go list ./...)

echo "-=Stop docker-compose=-"
cd tests
docker-compose down -v
