 # How to Reinit from version 0.1.4

This document describes how to make a signature using:
1. Version 2.0.0 or above,
2. Append log from version 0.1.4;
3. The Airgapped private key mnemonic that was saved during the master ceremony setup.

### Initial setup

1. Make sure that you have the Airgapped private key mnemonic;
2. Download the release binaries (`dc4bc_dkg_reinitializer`, `dc4bc_cli`, `dc4bc_d`, `index.tml`) for your platform from [the release page](https://github.com/lidofinance/dc4bc/releases/tag/2.0.0);
3. Download the old [append log dump](https://github.com/lidofinance/dc4bc/releases/download/2.0.0/dc4bc_async_ceremony_13_12_2020_dump.csv);
4. Set up your cold and hot nodes using the [instruction](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#setting-up-hot-and-airapped-nodes).

_Note that on latest macOS verssions the downloaded binaries might be marked as "quarantined". Usually there are two ways to mitigate that:_

* Right click on application and click "Open" from the context menu. There will be a warning, just click "Open". OSX will remember your choice and next time it will open;
* Remove 'quarantine attribute' from the app. In terminal run command: `xattr -d com.apple.quarantine <your_app>`.

### Generating new keys

Each participant must generate a new pair of communication keys for you Client node. This means that you **don't need any of the old states**. To do so, please refer to [this section](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#generating-keypairs-and-running-nodes). Since you're here, you must have done this before. The only difference this time is you don't need to backup the generated bip39 seed.

After starting the Airgapped machine, you must recover your private DKG key-pair using the mnemonic you got at the first initialisation:
```
>>> set_seed
> WARNING! this will overwrite your old seed, which might make DKGs you've done with it unusable.
> Only do this on a fresh db_path. Type 'ok' to  continue: ok
> Enter the BIP39 mnemonic for a random seed:
```

### Preparing the reinit Operation

All participants must now share their public communication keys. Run the command below to get your public communication key:
```
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
```

Someone must put those keys to `keys.json` in the following format and send that file to all participants (possibly on Discord):
```json
{
  "gergold": "8beWWNydtmEPISNXSC+Vp7U8nJrk23m9goW2hCX/eOo=",
  "svanevik": "r6cAAXor6iSy6nRipvLvfrNQe2BAVDQp8UN9Z+gZCNc=",
  "staking_facilities": "EYdeAeCJLNUXufn6SOFbdr0HWp3f0YwurHcVE2yGLvY=",
  "chorusone": "n/xcqib0t5bsyQ93Vfk4DXgahiEqrh4pEVKtWxH64UA=",
  "will": "qK3DvQ4S52/D8OLXDw3rue510A4GsEA7ZffVIuPoUyk=",
  "musalbas": "STFn4nporJXEPa+ftdztVo7zKa7Z4nG2eEV0jGJf2Mc=",
  "certus_one": "/QInqEyNBq0JZJuEQf1ZKLgQtTZJSdWJHr3KV5YAc0A=",
  "banteg": "ME06OxCRYZjz/sNv5mpBNRq2SHiVOAfmdaSyreSaEkk=",
  "k06a": "fK6y9yk06VNh8PZTwVQjEdyl40yta2aGlu0VXNYJfhM=",
  "rune": "v94jhyVQRRRaLiH4uxUljEeAC0uRzQERzCOsbU8lvYk=",
  "michwill": "FoiZGSPRFZTRZ98fTjjgeLs3fU8QRdXEadagcDW5zdY="
}
```

Everyone then must use the ```dc4bc_dkg_reinitializer``` utility (available with the `darwin` or `linux` suffix on the release page, see above) to generate a reinit message for `dc4bc_d`.

First you need to check the old [append log dump](https://github.com/lidofinance/dc4bc/releases/download/2.0.0/dc4bc_async_ceremony_13_12_2020_dump.csv) and the `keys.json` checksum:
```
shasum dc4bc_async_ceremony_13_12_2020_dump.csv
b9934eeb7abf7a5563ad2ad06ede87ff58c89b0c  dc4bc_async_ceremony_13_12_2020_dump.csv
shasum keys.json
9c08507c073642c0e97efc87a685c908e871ef8a  keys.json
```
If the checksums are correct for all participants, everyone should run:
```
./dc4bc_dkg_reinitializer reinit -i dc4bc_async_ceremony_13_12_2020_dump.csv -o reinit.json -k keys.json --adapt_0_1_4 --skip-header
```
In this example the message will be saved to ```reinit.json``` file.
* `--adapt_0_1_4`: this flag patches the old append log so that it is compatible with the latest version. You can see the utility source code [here](https://github.com/lidofinance/dc4bc/blob/eb72f74e25d910fc70c4a77158fed07435d48d7c/client/client.go#L679);
* `-k keys.json`: new communication public keys from this file will be added to `reinit.json`.

**All participants should run this command and check the `reinit.json` file checksum:**
```
./dc4bc_cli get_reinit_dkg_file_hash reinit.json
f65e4d87dce889df00ecebeed184ee601c23e531
```

### Running the reinit 

After everyone has generated the reinit.json file and verified the checksum, you must choose **one** participant that will prepare the reinit Operation for everyone. This participant must use the ```reinit_dkg``` command in dc4bc_cli to send the message to the append-only log:
```
$ ./dc4bc_cli reinit_dkg reinit.json
```
This command will send the message to the append-only log. The Client node process it and then will return an operation that must be handled like in the previous steps (scan GIF, go to an airgapped machine, etc.). **This step is for all participants.**

```
$ ./dc4bc_cli get_operations
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: d62c6c478d39d4239c6c5ceb0aea6792
                Operation ID: 34799e2301ae794c0b4f5bc9886ed5fa
                Description: reinit DKG
                Hash of the reinit DKG message - f65e4d87dce889df00ecebeed184ee601c23e531
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

There is a hash of the reinit DKG message in a reinitDKG operation and if it's not equal to the hash from ```get_reinit_dkg_file_hash``` command, that means that person who started the reinit process has changed some parameters.

Scan the operation JSON using the QR-scanning web-app and feed it to the airgapped machine via the security channel. The way to communicate via the security channel is well described in [this section](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#getting-familiar-with-the-secure-channel).

```
$ >>> read_operation
> Enter the path to Operation JSON file: /tmp/dkg_id_dc1dd_step_0_reinit_DKG_f420a_request.json
```

Now you have your master DKG pubkey recovered, and you can sign new messages! Signing algorithm after DKG reinitialisation is just the same as in the usual flow, simply follow the [instruction](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#signature).
