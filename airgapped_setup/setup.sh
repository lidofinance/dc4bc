#!/bin/bash

wget -q --show-progress https://ftp.mozilla.org/pub/firefox/releases/83.0/linux-x86_64/en-US/firefox-83.0.tar.bz2
shasum firefox-83.0.tar.bz2 |
while read -r sum _ ; do
  [[ $sum == 43a0fb57ce1bfd348a15bbcc092ac7cced79ce79 ]] && echo "Firefox checksum is correct." || echo "[WARN] Incorrect Firefox checksum!"
done

wget -q --show-progress https://github.com/lidofinance/dc4bc/releases/download/0.1.1/dc4bc_airgapped_linux
shasum dc4bc_airgapped_linux |
while read -r sum _ ; do
  [[ $sum == a15b791e3926f9925e560ef813d9fe3e30d777f5 ]] && echo "Airgapped checksum is correct." || echo "[WARN] Incorrect Airgapped checksum!"
done

cp ../qr_reader_bundle/index.html ./index.html
mv dc4bc_airgapped_linux dc4bc_airgapped
