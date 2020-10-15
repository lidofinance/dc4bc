#!/bin/sh
set -e

PASSWORD=test1234

# Creating TLS CA, Certificates and keystore / truststore
rm -rf certs
mkdir -p certs
# Generate CA certificates
openssl req -new -nodes -x509 -days 3650 -newkey rsa:2048 -keyout certs/ca.key -out certs/ca.crt -config ca.cnf
cat certs/ca.crt certs/ca.key > certs/ca.pem

# Generate kafka server certificates
openssl req -new -newkey rsa:2048 -keyout certs/server.key -out certs/server.csr -config server.cnf -nodes
openssl x509 -req -days 3650 -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -extfile server.cnf -extensions v3_req
openssl pkcs12 -export -in certs/server.crt -inkey certs/server.key -chain -CAfile certs/ca.pem -name "kafka.confluent.local" -out certs/server.p12 -password pass:$PASSWORD

# Import server certificate to keystore and CA to truststore
keytool -importkeystore -deststorepass $PASSWORD -destkeystore certs/server.keystore.jks \
    -srckeystore certs/server.p12 \
    -deststoretype PKCS12  \
    -srcstoretype PKCS12 \
    -noprompt \
    -srcstorepass $PASSWORD

keytool -keystore certs/truststore.jks -alias CARoot -import -file certs/ca.crt -storepass $PASSWORD  -noprompt -storetype PKCS12

# Starting docker-compose services
docker-compose up -d --build