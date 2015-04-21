#!/bin/bash

# This is just a script for generating a self signed HTTPs certificate

command -v openssl >/dev/null 2>&1 || { echo >&2 "This script requires openssl. Aborting."; exit 1; }

openssl req -new > tmp.csr
openssl rsa -in privkey.pem -out certificate.key
openssl x509 -in tmp.csr -out certificate.cert -req -signkey certificate.key

# Clean up

rm -f tmp.csr privkey.pem
