#!/usr/bin/env bash

echo "download tampermonkey addon"
wget -q --show-progress --progress dot:binary -O tampermonkey.xpi https://addons.mozilla.org/firefox/downloads/file/4405733/tampermonkey-5.3.3.xpi
mv tampermonkey.xpi tampermonkey.zip
unzip -q tampermonkey.zip -d tampermonkey

echo "done"
