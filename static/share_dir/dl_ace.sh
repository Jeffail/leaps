#!/bin/bash

ACE_VERSION="1.1.7"
ACE_ARCHIVE="ace-builds-${ACE_VERSION}"

mkdir -p ./ace

wget https://github.com/ajaxorg/ace-builds/archive/v${ACE_VERSION}.zip -O tmp_ace.zip
unzip "./tmp_ace.zip"

( cd "./${ACE_ARCHIVE}" && rsync -a ./src-min/ ../ace )

rm -rf "./${ACE_ARCHIVE}"
rm "./tmp_ace.zip"
