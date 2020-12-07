#!/bin/bash

wget -q --show-progress https://ftp.mozilla.org/pub/firefox/releases/83.0/linux-x86_64/en-US/firefox-83.0.tar.bz2
shasum firefox-83.0.tar.bz2 |
while read -r sum _ ; do
  [[ $sum == 19asdasdasd56462e44d61a093ea57e964cf0af05c0e ]] && echo "Firefox checksum is correct." || echo "[WARN] Incorrect Firefox checksum!"
done

wget -q --show-progress https://github.com/lidofinance/dc4bc/releases/download/0.1.0/dc4bc_airgapped_linux
shasum dc4bc_airgapped_linux |
while read -r sum _ ; do
  [[ $sum == 19asdasdasd56462e44d61a093ea57e964cf0af05c0e ]] && echo "Airgapped checksum is correct." || echo "[WARN] Incorrect Airgapped checksum!"
done

cp ../qr_reader_bundle/index.html ./index.html
mv dc4bc_airgapped_linux dc4bc_airgapped
