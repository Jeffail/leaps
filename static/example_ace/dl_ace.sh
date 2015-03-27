#!/bin/bash

ACE_VERSION="1.1.8"
ACE_ARCHIVE="ace-builds-${ACE_VERSION}"

mkdir -p ./ace

wget https://github.com/ajaxorg/ace-builds/archive/v${ACE_VERSION}.tar.gz -O tmp_ace.tar.gz
tar -xvf "./tmp_ace.tar.gz"

( cd "./${ACE_ARCHIVE}" && rsync -a ./src-min/ ../ace )

rm -rf "./${ACE_ARCHIVE}"
rm "./tmp_ace.tar.gz"
