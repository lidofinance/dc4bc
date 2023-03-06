#!/bin/bash

wget -q --show-progress https://ftp.mozilla.org/pub/firefox/releases/110.0.1/linux-x86_64/en-US/firefox-110.0.1.tar.bz2
sha256sum firefox-110.0.1.tar.bz2 |
while read -r sum _ ; do
  [[ $sum == 0ffd6499f8e2bb31d5321a6ec1ed5c2fcfb22f917a49a2b0c2d8b6fd379a1e7f ]] && echo "Firefox checksum is correct." || echo "[WARN] Incorrect Firefox checksum!"
done

wget -q --show-progress https://github.com/lidofinance/dc4bc/releases/download/4.0.0/build-linux-amd64.tar
tar -xvf build-linux-amd64.tar
shasum ./build/dc4bc_airgapped |
while read -r sum _ ; do
  [[ $sum == 6508c7fd3b055d90f0725b188a59ebf7060255b3 ]] && echo "Airgapped checksum is correct." || echo "[WARN] Incorrect Airgapped checksum!"
done

cp ../qr_reader_bundle/qr-tool.html ./qr-tool.html
mv ./build/dc4bc_airgapped dc4bc_airgapped