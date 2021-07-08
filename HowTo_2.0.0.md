 # How to Reinit from version 1.4.0

This document describes how to make a signature using:
1. Version 2.0.0,
2. Append log from version 1.4.0;
3. The Airgapped private key mnemonic that was saved during the master ceremony setup.

#### Initial setup

1. Make sure that you have the Airgapped private key mnemonic;
2. Download the release binaries (`dc4bc_dkg_reinitializer`, `dc4bc_airgapped`, `dc4bc_cli`, `dc4bc_d`, `dc4bc_dkg_reinit_log_adapter`) with appropriate platform suffix) from [the release page](https://github.com/lidofinance/dc4bc/releases/tag/2.0.0);
3. Set up your cold and hot nodes using the old [instruction](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#setting-up-hot-and-airapped-nodes).

#### 1.4.0 -> 2.0.0 Migration

Each participant must generate a new pair of communication keys for you Client node. This means that you **don't need any of the old states**:
```
$ ./dc4bc_d gen_keys --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store
```
After you have the keys, start the node:
```
$ ./dc4bc_d start --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store --state_dbdsn ./stores/dc4bc_<YOUR USERNAME>_state --listen_addr localhost:8080 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_dbdsn 51.158.98.208:9093 --storage_topic <DKG_TOPIC> --kafka_consumer_group <YOUR USERNAME>_group
```
* `--username` — This username will be used to identify you during DKG and signing
* `--key_store_dbdsn` — This is where the keys that are used for signing messages that will go to the Bulletin Board will be stored. Do not store these keys in `/tmp/` for production runs and make sure that you have a backup
* `--state_dbdsn` This is where your Client node's state (including the FSM state) will be kept. If you delete this directory, you will have to re-read the whole message board topic, which might result in odd states
* `--storage_dbdsn` This argument specifies the storage endpoint. This storage is going to be used by all participants to exchange messages
* `--storage_topic` Specifies the topic (a "directory" inside the storage) that you are going to use. Typically participants will agree on a new topic for each new signature or DKG round to avoid confusion
* `--kafka_consumer_group` Specifies your consumer group. This allows you to restart the Client and read the messages starting from the last one you saw.
**Note that you have to set the `kafka_consumer_group` (this was not present in 1.4.0). The recommended value for user Alice would be `alice_group`.**

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

**After everyone has set up their Client and Airgapped machines, you must choose one participant (Bob) that will prepare the reinit Operation for everyone.**

Bob must use the ```dc4bc_dkg_reinitializer``` utility (available with the `darwin` or `linux` suffix on the release page, see above) to generate a reinit message for dc4bc_d:

```shell
$ ./dc4bc_dkg_reinitializer reinit --storage_dbdsn 51.158.98.208:9093 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_topic <OLD_DKG_TOPIC> --kafka_consumer_group <YOUR USERNAME>_group -o reinit.json
```
In this example the message will be saved to ```reinit.json``` file. The <OLD_DKG_TOPIC> is the topic with the messages from the old DKG; it will be provided by the Lido team.

All participants must share their public communication keys. Now Bob needs to open ```reinit.json``` and paste the shared communication public keys in relevant "new_comm_pub_key" fields:

```
{
  "dkg_id": "d62c6c478d39d4239c6c5ceb0aea6792",
  "threshold": 2,
  "participants": [
    {
      "dkg_pub_key": "mY+odV9fqJA7TRNUrqQd29f3CmJoY/ug6mwG5CbnHdiHdmjHHGNjWgo/GXWiJ5sa",
      "old_comm_pub_key": "ythrIEb8zqO56hO/SIeCXp0QIzV96VVg9nc6yIrIFdQ=",
      "new_comm_pub_key": null, <<< PASTE HERE
      "name": "alice"
    },
    {
      "dkg_pub_key": "jx71Coo+EDkzWnAZLwRFX8rnJH5gwhTdldWHfNvK4NiOLixHck0JOepkgtjoNcRb",
      "old_comm_pub_key": "8ogpX6PPWIS8Unx0XKC7gcEypPaJZdwVPbE24dVoRNo=",
      "new_comm_pub_key": null, <<< PASTE HERE
      "name": "bob"
    }
  ],
  ...
```

Now, to make the old append log from ```reinit.json``` compatible with the 2.0.0 release, Bob needs to patch it by running the `dkg_reinit_log_adapter` utility with original old log file as the first argument and a name for the new patched file as the second argument:
```bash
./dc4bc_dkg_reinit_log_adapter reinit.json new_reinit.json 
```
You can find the utility source code [here](https://github.com/lidofinance/dc4bc/tree/master/cmd/dkg_reinit_log_adapter).

Then Bob must use the ```reinit_dkg``` command in dc4bc_cli to send the message to the append-only log:

```shell
$ ./dc4bc_cli reinit_dkg reinit.json
```

The command will send the message to the append-only log. The Client node process it and then will return an operation that must be handled like in the previous steps (scan GIF, go to an airgapped machine, etc.). **This step is for all participants, not only for Bob.**

```
$ ./dc4bc_cli get_operations
Please, select operation:
-----------------------------------------------------
 1)		DKG round ID: d62c6c478d39d4239c6c5ceb0aea6792
		Operation ID: 34799e2301ae794c0b4f5bc9886ed5fa
		Description: reinit DKG
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

After you have processed the operation in airgapped, you have your master DKG pubkey recovered, so you can sign new messages!

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
