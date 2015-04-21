#!/bin/bash

# This is just a script for generating a .htpasswd file with SHA hashes, or adding accounts to an existing file

if [ -z "$1" ]; then
	echo "usage: $0 <name-of-user>"
	exit 1
fi

command -v htpasswd >/dev/null 2>&1 || { echo >&2 "This script requires htpasswd (sudo apt-get install apache2-utils). Aborting."; exit 1; }

if [ -f .htpasswd ]; then
	htpasswd -s .htpasswd $1
else
	htpasswd -cs .htpasswd $1
fi
