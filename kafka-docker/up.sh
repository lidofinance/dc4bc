#!/bin/sh
set -e

PASSWORD=test1234

rm -rf certs
mkdir -p certs

# self sign cert
openssl req -new -nodes -x509 -days 3650 -newkey rsa:2048 -keyout certs/ca.key -out certs/ca.crt -config ca.cnf
cat certs/ca.crt certs/ca.key > certs/ca.pem

openssl pkcs12 -export -in certs/ca.crt -inkey certs/ca.key -out certs/server.p12 -password pass:$PASSWORD

keytool -importkeystore -deststorepass $PASSWORD -destkeystore certs/server.keystore.jks \
    -srckeystore certs/server.p12 \
    -deststoretype PKCS12  \
    -srcstoretype PKCS12 \
    -noprompt \
    -srcstorepass $PASSWORD

keytool -keystore certs/truststore.jks -alias CARoot -import -file certs/ca.crt -storepass $PASSWORD  -noprompt -storetype PKCS12

docker-compose up -d --build