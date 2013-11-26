#!/bin/sh
openssl x509 -in $1 -inform DER -out $2 -outform PEM
