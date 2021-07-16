 # How to Reinit from version 1.4.0

This document describes how to make a signature using:
1. Version 2.0.0,
2. Append log from version 1.4.0;
3. The Airgapped private key mnemonic that was saved during the master ceremony setup.

#### Initial setup

1. Make sure that you have the Airgapped private key mnemonic;
2. Download the release binaries (`dc4bc_dkg_reinitializer`, `dc4bc_airgapped`, `dc4bc_cli`, `dc4bc_d`, `index.tml`) for your platform from [the release page](https://github.com/lidofinance/dc4bc/releases/tag/2.0.0);
3. Download the old [append log dump](https://github.com/lidofinance/dc4bc/releases/download/2.0.0/dc4bc_async_ceremony_13_12_2020_dump.csv);
4. Set up your cold and hot nodes using the old [instruction](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#setting-up-hot-and-airapped-nodes).

_Note that on latest macOS verssions the downloaded binaries might be marked as "quarantined". Usually there are two ways to mitigate that:_

* Right click on application and click "Open" from the context menu. There will be a warning, just click "Open". OSX will remember your choice and next time it will open;
* Remove 'quarantine attribute' from the app. In terminal run command: `xattr -d com.apple.quarantine <your_app>`.

#### Generating new keys

Each participant must generate a new pair of communication keys for you Client node. This means that you **don't need any of the old states**:
```
$ ./dc4bc_d gen_keys --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store
```
After you have the keys, start the node:
```
$ ./dc4bc_d start --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store --state_dbdsn ./stores/dc4bc_<YOUR USERNAME>_state --listen_addr localhost:8080 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_dbdsn 94.130.57.249:9093 --storage_topic <DKG_TOPIC> --kafka_consumer_group <YOUR USERNAME>_group
```
* `--username` — This username will be used to identify you during DKG and signing;
* `--key_store_dbdsn` — This is where the keys that are used for signing messages that will go to the Bulletin Board will be stored. Do not store these keys in `/tmp/` for production runs and make sure that you have a backup;
* `--state_dbdsn` This is where your Client node's state (including the FSM state) will be kept. If you delete this directory, you will have to re-read the whole message board topic, which might result in odd states;
* `--storage_dbdsn` This argument specifies the storage endpoint. This storage is going to be used by all participants to exchange messages;
* `--storage_topic` Specifies the topic (a "directory" inside the storage) that you are going to use. Typically participants will agree on a new topic for each new signature or DKG round to avoid confusion.

Then start the Airgapped machine:
```
$ ./dc4bc_airgapped --db_path ./stores/dc4bc_<YOUR USERNAME>_airgapped_state --password_expiration 10m
```
After starting the Airgapped machine, you must recover your private DKG key-pair using the saved mnemonic:

```shell
>>> set_seed
> WARNING! this will overwrite your old seed, which might make DKGs you've done with it unusable.
> Only do this on a fresh db_path. Type 'ok' to  continue: ok
> Enter the BIP39 mnemonic for a random seed:
```

#### 1.4.0 -> 2.0.0 Migration

**After everyone has set up their Client and Airgapped machines, you must choose one participant (Bob) that will prepare the reinit Operation for everyone.**

All participants must now share their public communication keys. Run the command below to get your public communication key:
```
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
```
Bob must put those keys to `keys.json` in the following format and send that file to all participants:
```
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

Bob then must use the ```dc4bc_dkg_reinitializer``` utility (available with the `darwin` or `linux` suffix on the release page, see above) to generate a reinit message for `dc4bc_d`.

First everyone needs to check the old [append log dump](https://github.com/lidofinance/dc4bc/releases/download/2.0.0/dc4bc_async_ceremony_13_12_2020_dump.csv) and the `keys.json` checksum:
```
shasum dc4bc_async_ceremony_13_12_2020_dump.csv
b9934eeb7abf7a5563ad2ad06ede87ff58c89b0c  dc4bc_async_ceremony_13_12_2020_dump.csv
shasum keys.json
9c08507c073642c0e97efc87a685c908e871ef8a  keys.json
```
If the checksum is correct for all participants, everyone should run:
```shell
./dc4bc_dkg_reinitializer reinit -i dc4bc_async_ceremony_13_12_2020_dump.csv -o reinit.json -k keys.json --adapt_1_4_0 --skip-header
```
In this example the message will be saved to ```reinit.json``` file.
* `--adapt_1_4_0`: this flag patches the old append log so that it is compatible with the latest version. You can see the utility source code [here](https://github.com/lidofinance/dc4bc/blob/eb72f74e25d910fc70c4a77158fed07435d48d7c/client/client.go#L679);
* `-k keys.json`: new communication public keys from this file will be added to `reinit.json`.

**All participants should run this command and check the `reinit.json` file checksum:**
```
./dc4bc_cli get_reinit_dkg_file_hash reinit.json
f65e4d87dce889df00ecebeed184ee601c23e531
```
Then Bob must use the ```reinit_dkg``` command in dc4bc_cli to send the message to the append-only log:
```shell
$ ./dc4bc_cli reinit_dkg reinit.json
```
This command will send the message to the append-only log. The Client node process it and then will return an operation that must be handled like in the previous steps (scan GIF, go to an airgapped machine, etc.). **This step is for all participants, not only for Bob.**

```
$ ./dc4bc_cli get_operations
Please, select operation:
-----------------------------------------------------
 1)		DKG round ID: d62c6c478d39d4239c6c5ceb0aea6792
		Operation ID: 34799e2301ae794c0b4f5bc9886ed5fa
		Description: reinit DKG
		Hash of the reinit DKG message - f65e4d87dce889df00ecebeed184ee601c23e531
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

There is a hash of the reinit DKG message in a reinitDKG operation and if it's not equal to the hash from ```get_reinit_dkg_file_hash``` command, that means that person who started the reinit process has changed some parameters.

Scan the operation using the QR-scanning web-app (open `index.html` in your browser). Now you need to process the operation inside the Airgapped machine:

```
$ >>> read_operation
> Enter the path to Operation JSON file: reinit_operation.json
```

Now you have your master DKG pubkey recovered, and you can sign new messages!

#### Signature

Now we have to collectively sign a message. Some participant will run the command that sends an invitation to the message board:

```
# Inside dc4bc_airgapped prompt:
$ >>> show_finished_dkg
AABB10CABB10
$ echo "the message to sign" > data.txt
$ ./dc4bc_cli sign_data AABB10CABB10 data.txt --listen_addr localhost:8080
```
Further actions are repetitive and are similar to the DKG procedure. Check for new pending operations, feed them to `dc4bc_airgapped`, pass the responses to the client, then wait for new operations, etc. After some back and forth you'll see the node tell you that the signature is ready:
```
[john_doe] Handling message with offset 40, type signature_reconstructed
Successfully processed message with offset 40, type signature_reconstructed
```

Now you have the full reconstructed signature.
```
./dc4bc_cli get_signatures AABB10CABB10
Signing ID: 909b7660-ccc4-45c4-9201-e30015a69425
	DKG round ID: AABB10CABB10
	Participant: john_doe
	Reconstructed signature for the data: tK+3CV2CI0flgwWLuhrZA5eaFfuJIvpLAc6CbAy5XBuRpzuCkjOZLCU6z1SvlwQIBJp5dAVa2rtbSy1jl98YtidujVWeUDNUz+kRl2C1C1BeLG5JvzQxhgr2dDxq0thu
```
It'll show you a list of broadcasted reconstructed signatures for a given DKG round.

You can verify any signature by executing `verify_signature` command inside the airgapped prompt:
```
>>> verify_signature
> Enter the DKGRoundIdentifier: AABB10CABB10
> Enter the BLS signature: tK+3CV2CI0flgwWLuhrZA5eaFfuJIvpLAc6CbAy5XBuRpzuCkjOZLCU6z1SvlwQIBJp5dAVa2rtbSy1jl98YtidujVWeUDNUz+kRl2C1C1BeLG5JvzQxhgr2dDxq0thu
> Enter the message which was signed (base64): dGhlIG1lc3NhZ2UgdG8gc2lnbgo=
Signature is correct!
```

Now the ceremony is  over. 
