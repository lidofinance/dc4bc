# dc4bc threshold signature distributed key generation step-by-step guide

You'll generate key shards for a threshold signature potentially holding multimillion balance. We designed a pretty complicated software to handle that; it's a bit rough around the edges but serviceable. It's using a hot internet-connected machine for communication and an airgapped machine for handling secrets; we'd prefer to use hardware wallet for this but it wasn't possible on our timeline. 

## Large-scale steps:

0. Build or download software
1. Setup hot and airgapped machines, backup media 
2. Generate participant's keypairs
3. Publish your public keys and agree on threshold signature participant set
4. Initiate dkg ceremony
5. Confirm ceremony participation
6. Broadcast commits
7. Collect commits and broadcast deals
8. Collect deals and broadcast reconstructed public key
9. Do a test run of threshold signing a message
10. Check signature correctness

### Building software

Clone the project repository:
```
git clone https://github.com/lidofinance/dc4bc.git
```

#### Installation (Linux)

First install the Go toolchain:
```shell
curl -OL https://golang.org/dl/go1.15.2.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.15.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Then build the project binaries:
```shell
# Go to the cloned repository.
cd dc4bc
make build
```

#### Installation (Darwin)

First install the Go toolchain:
```shell
mkdir $HOME/Go
export GOPATH=$HOME/Go
export GOROOT=/usr/local/opt/go/libexec
export PATH=$PATH:$GOPATH/bin
export PATH=$PATH:$GOROOT/bin
brew install go
```

Then build the project binaries:
```shell
# Go to the cloned repository.
cd dc4bc
make build
```

#### QR encoder/decoder

Get the latest release of the web interface based QR encoder/decoder from the following repository:
```
https://github.com/lidofinance/qr-tool/releases
```
Or simply use one of its versions located at ./qr_reader_bundle/qr-tool.html.

### Downloading

Check out project releases tab in github and get the distribuition binaries for your system. Also clone the repository anyway, because you'll need the certificate file for kafka that is not a part of the releases files.

#### Setting up hot and airapped nodes

Following is a reasonably secure setup for an airgapped machine. It does not account for supply chain attacks (someone shipping you posined hardware) and wireless physical proximity 0day exploits but otherwise is reasonably good. With good the geographical and otherwise diversity of co-signers this should be good enough to make collusion the main practical avenue of attack.

Setup:

Hot node: linux or os x machine with webcam. Probably a laptop. Setup doesn't really matter as long as it's your machine (not shared or rented) and you're reasonably sure it's not compromised. Should have a reliable network connection during the ceremony. It stores communication keys that do not protect any critical secrets.

Airgapped node: 

A laptop that can run Tails (linux distribution) and has a webcam. Preferably it has hardware switch for wireless connections or, even better, wireless chips removed. Shouldn't have wireless keyboard or mouse adapters plugged in either.

NB: If you know what you're doing set up an airgapped machine yourself or use one you already have. Tails live dvd/usb is not the best possible setup - it's just good enough in our opinion.

Backup media: paper wallet or another media you will use to backup bip39 word-based seed. You will keep this backup until withdrawals are enabled in eth2.

Plaintext media: 1+ gb usb drive or cd/dvd for non-secrets (executables and the like). If you choose USB drive, you'll be disposing of that particular drive by the end of a ceremony.

1. Make a bootable media for tails using https://tails.boum.org/install/index.en.html instructions. Live dvd is preferabe to a usb stick but usb stick is a valid option. Verification of an image signature per instructions on Tail's site is strongly recommended. 
2. Prepare a plaintext media: run a script `sh airapped_folder/setup.sh` to set up a folder with one on your hot node, then copy/burn that folder on the plaintext media. It contains airapped node binary, firefox distribution, qr code reader html file and deploy script to easily copy all that to tails distribution.
3. If your airgapped laptop allows it, switch the wireless hardware off. Boot into Tails, selecting "no network connection" as an additional option on starting (always select this option on an airgapped machine). From this point on to the end of ceremony the machine shouldn't ever connect to the network; if you can afford having it permanently airgapped forever - can by useful in crypto - do it.
4. Insert a plaintext media and run `sh ./deploy.sh` from there to copy all the needed files to your ephemeral home dir on tails.

Now you've got all paraphernalia set up and can proceed with the guide further.

### DKG

The goal of DKG is to produce a set of secrets, and those secrets can be potentially used for managing vast amounts of money. Threfore it is obvious that you would like your private key share to be generated and stored as securely as possible. To achieve the desired security level, you should have access to two computers: one for the Client node (with a web camera and with access to the Internet) and one for the Airgapped machine (just with a web camera).

#### Generating keypairs and running nodes

To start a DKG round, you should first generate two pairs of keys: one pair is for signing messages that will go to the Bulletin Board, and the other one will be used by the Airgapped Machine to encrypt private messages (as opposed to the messages that are broadcasted).

##### Starting the hot node

First, generate keys for your Client node:
```
$ ./dc4bc_d gen_keys --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store
```
Immediately backup the key store: these keys won't be the ones to hold money, but if they are lost during the initial ceremony dkg round will have to be restarted.

After you have the keys, start the node:
```
$ ./dc4bc_d start --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store --state_dbdsn ./stores/dc4bc_<YOUR USERNAME>_state --listen_addr localhost:8080 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_dbdsn 94.130.57.249:9093 --storage_topic <DKG_TOPIC> --kafka_consumer_group <YOUR USERNAME>_group
```
* `--username` — This username will be used to identify you during DKG and signing
* `--key_store_dbdsn` — This is where the keys that are used for signing messages that will go to the Bulletin Board will be stored. Do not store these keys in `/tmp/` for production runs and make sure that you have a backup;
* `--state_dbdsn` This is where your Client node's state (including the FSM state) will be kept. If you delete this directory, you will have to re-read the whole message board topic, which might result in odd states;
* `--storage_dbdsn` This argument specifies the storage endpoint. This storage is going to be used by all participants to exchange messages;
* `--storage_topic` Specifies the topic (a "directory" inside the storage) that you are going to use. Typically participants will agree on a new topic for each new signature or DKG round to avoid confusion;
* `--kafka_consumer_group` Specifies your consumer group. This allows you to restart the Client and read the messages starting from the last one you saw.

##### Starting the aigrapped machine

Then start the airgapped machine:
```
$ ./dc4bc_airgapped --db_path ./stores/dc4bc_<YOUR USERNAME>_airgapped_state --password_expiration 10m
```
* `--db_path` Specifies the directory in which the Aigapped machine state will be stored. If the directory that you specified does not exist, the Airgapped machine will generate new keys for you on startup. *N.B.: It is very important not to put your Airgapped machine state to `/tmp` or to occasionally lose it. Please make sure that you keep your Airgapped machine state in a safe place and make a backup.*
* `--password_expiration` Specifies the time in which you'll be able to use the Airgapped machine without re-entering your password. The Airgapped machine will ask you to create a new password during the first run. Make sure that the password is not lost.

Backup the generated bip39 seed on a paper wallet; if you need to restore it, use the `set_seed` command in the airgapped executable's console.

##### Sharing the keys

Print your communication public key and encryption public key. *You will have to publish them during the [Conference call](https://github.com/lidofinance/dc4bc-conference-call) along with the `--username` that you specified during the Client node setup).*
```
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
# Inside the airgapped shell:
>>> show_dkg_pubkey
sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
>>> generate_dkg_pubkey_json
A JSON file with DKG public key was saved to: /tmp/dc4bc_json_dkg_pub_key.json
```

**N.B.: You can start and stop both the Client node and the Airgapped machine any time you want given that the states are stored safely on your computer. When you restart the Airgapped machine, make sure that you run the `replay_operations_log` command exactly once before performing any actions — that will make the Airgapped machine replay the state and be ready for new actions. Please do not replay the log more than once during one Airgapped session, this might lead to undefined state.**

#### Invitation to DKG

Now you want to start the DKG procedure. *This action must be done exactly once by only one of the participants. The participants must decide who will send the initial message collectively.* 

Tell the node to send an InitDKG message that proposes to run DKG with parameters which are located in a `start_dkg_propose.json` file. This file is created collectively during a [Conference call](https://github.com/lidofinance/dc4bc-conference-call) by the participants.
```
$ ./dc4bc_cli start_dkg /path/to/start_dkg_propose.json --listen_addr localhost:8080
```
Example of start_dkg_propose.json file structure:
```json
{
  "SigningThreshold": 2,
  "Participants": [
    {
      "Username": "user1",
      "PubKey": "GW7lJ6ojeOQYtIKU6y3/LghSFo1N9Rq5rGBwMRKRl8o=",
      "DkgPubKey": "hV+MxO1iPIWH/T3Y4kspQTE0uLzYsaTcWK88bGbhoVYBGuP59hs/jhrhkTvDiF2y"
    },
    {
      "Username": "user2",
      "PubKey": "WCOSdutt0SkbYiNy/XENfvx1fYErM3Y41I517fHhX9c=",
      "DkgPubKey": "p00s39o7fsWBu5ldRW+VBeMJup3SGMw4jLIg79lAXRG+D6UIF8mw5M4dTVgB58g7"
    }
  ]
}
```

The message will be consumed by your node:
```
[john_doe] starting to poll messages from append-only log...
[john_doe] Starting HTTP server on address: localhost:8080
[john_doe] Handling message with offset 0, type event_sig_proposal_init
[john_doe] message event_sig_proposal_init done successfully from john_doe
[john_doe] Successfully processed message with offset 0, type event_sig_proposal_init
```

Now you have a pending operation in your operation pool. That is an operation to confirm participation: you'll be bringing dkg setup data to the airgapped machine and submitting the message that you will participate. 

Get the list of pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Operation ID: 83e14a21c0116094630654d923ba9600
                Description: confirm participation in the new DKG round
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

You can check the hash of the proposing DKG message:
```
./dc4bc_cli get_start_dkg_file_hash start_dkg_propose.json
a60bd47a831cd58a96bdd4381ee15afc
```
The command returns a hash of the proposing message. If it is not equal to the hash from the list of pending operations, that means the person who proposed to start the DKG round changed the parameters that you agreed on the Conferce Call.

Select an operation by typing its number and press Enter. This operation requires only a confirmation of user, so it's just sends a message to the append-only log.
```
[john_doe] message event_sig_proposal_confirm_by_participant done successfully from john_doe
```

When all participants confirm their participation in DKG round, the node will proceed to the next step.

#### Distributed key generation ceremony

Once confirmations are sent by all participants, you'll have a new operation:

```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Operation ID: df482be9eb1e50b0968a5daf7e52e073
                Description: send commits for the DKG round
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

##### Getting familiar with the secure channel

It's time to establish a secure connection between the machines. Select an operation to make the node produce a JSON file for it:
```
json file was saved to: /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_request.json
```

Open the [qr tool](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#qr-encoderdecoder) in your Web browser on the hot node machine and airgapped machine, pull your JSON file to the encoder and save the *.gif file.

On the airgapped machine open the decoder section and allow the page to use your camera. Show the animation from the hot node to the airgapped machine and wait until the QR code decoded back to a JSON.

Now go to `dc4bc_airgapped` prompt and enter the path to the file that contains the Operation JSON:

```
>>> read_operation
> Enter the path to Operation JSON file: /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_request.json
Operation JSON was handled successfully, the result Operation JSON was saved to: /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_result.json
```

Encode the result JSON file to a QR GIF on the airgapped machine and show the animation to the hot node machine. Then go to the node, decode GIF to JSON and run the following command using the path to the decoded json:
```
$ ./dc4bc_cli read_operation_result --listen_addr localhost:8080 /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_result.json
```
```
[john_doe] message event_dkg_commit_confirm_received done successfully from john_doe
```

##### Following up the ceremony

When all participants perform the necessary operations, the node will proceed to the next step. The next steps are:

- Broadcast commits - you'll be broadcasting a public derivative of your secret seed for the key shards that will be used to check that you don't try to cheat at DKG
- Collect commits and broadcast deals - you'll be collecting each other's commits and sending each participant a private message that will be used to construct your key shard
- Collect deals and broadcast reconstructed public key - you'll be using private messages from other people to generate your key shard

Further actions are repetitive. For each step check for new pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Operation ID: 988edba605afc4262827665f7d9395bf
                Description: send deals for the DKG round
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

Then feed them to `dc4bc_airgapped`, then pass the responses to the client, then wait for new operations, etc. After some back and forth you'll see the node tell you that DKG is finished (`event_dkg_master_key_confirm_received`):
```
[john_doe] State stage_signing_idle does not require an operation
[john_doe] Successfully processed message with offset 10, type event_dkg_master_key_confirm_received
```

To take a look at the DKG result, run the show_finished_dkg command in the dc4bc_airgapped:

```
$ >>> show_finished_dkg
DKG identifier: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
PubKey: lTRw8FNa8lGsUBDa1pT/DxzVVpnkFpwtpro6FRnSAl9YL3k5v/ira+2DepmjCIYQ
-----------------------------------------------------
```

Key generation ceremony is over. `exit` airgapped dkg tool prompt and backup your airapped machine db multiple times to different media. 

To check the backup run `dc4bc_airgapped -db_path /path/to/backup` and run `show_dkg_pubkey` command. If it works the backup is correct.

### Signature

Now we have to collectively sign a message. Some participant will run the command that sends an invitation to the message board:
```shell
$ echo "the message to sign" > data.txt
$ ./dc4bc_cli sign_data c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2 data.txt --listen_addr localhost:8080
```

**Note: if you want to sign a batch of messages, create a new directory, put all messages in separate files in that directory and use the `./dc4bc_cli sign_batch_data [dkg_id] [messages_dir]` command.**

As the result, all participants will get a new operation suggesting them to partially sign the proposed message:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Operation ID: 6a42609eb81c6a733d8ff4bb511b0d35
                Description: send your partial sign for the message
                Hash of the data to sign - 4a80926796c7646c1a50accf0477ea4ffb2e3ad55eacf083ce7ca472c4219bbf
                Batch ID: 1ad6a966-64d1-4a1a-ad96-022790cf57f0
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```
But before signing, spend some time on taking a look at what you're about to sign.

#### Checking the messages to sign content

At this point the other participants would probably like to take a look at the message they've been proposed to sign. To do so, they can run the following command that reveals a list of all messages related to a given DKG round as well as the messages signing IDs and hashes.
```
./dc4bc_cli get_signatures c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2 --json_files_folder /tmp/
json file was saved to: /tmp/dkg_signatures_dump_cc1e5.json
```

You can view the file with:

```
cat /tmp/dkg_signatures_dump_cc1e5.json | jq
{
  "665b9621-8fd0-454c-8294-c9466f5dce8f": {
    "payload_base64": "bWVzc2FnZSAyCg==",
    "signature": "sXQD+89/6+dtR7vuSFWK4DERFD1ygEvkA/AcYhKj1L/TRWARzhR7lj/i0qCwY8aDDRnEiEihZsXpIMwFnopeycnAhmAcBDyf2Mekpbc3Vrim9RCcNrxFqzHGTFC95kqD"
  },
  "c9e50034-112d-46c5-ad64-e718dccf8dd6": {
    "payload_base64": "bWVzc2FnZSAxCg==",
    "signature": "kWOAJ2QejehdUkMkOn3qhW430fcxrc2wdS6vlxpP9fOrTzYDgjCWWZtRJFfUILpxFOB5IWgQEI/BC/uDJM4AZNEX4tjucmgwx37hjMaE3qbc/rtS59IjLBnbeNYdM9ae"
  }
}
```

Then it's possible to reveal a message data by running the following command:
```
./dc4bc_cli get_signature_data c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2 ca800cac-2c13-4a14-8ca3-72c36112c5e4
the message to sign
```

#### Signing the message

Further steps are similar to the DKG procedure. First, select the pending `send your partial sign for the message` operation, feed it to `dc4bc_airgapped`, pass the response to the client, then wait until other participants do the same. Once the number of participants which signed the message is >= than the threshold, you'll see the cli `get_operations` tell you that the signature is ready to be reconstructered on the airgapped:
```
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Operation ID: 47b60d03f1922c614cb497705c56462a
                Description: recover full signature for the message
                Hash of the data to sign - 4a80926796c7646c1a50accf0477ea4ffb2e3ad55eacf083ce7ca472c4219bbf
                Signing ID: 1ad6a966-64d1-4a1a-ad96-022790cf57f0
```

Before that, it's possible to check the progress of signatures gathering and see who's already sent partial signs, and who hasn't:
```
./dc4bc_cli show_fsm_status c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
FSM current status is state_signing_await_partial_signs
Waiting for data from: jane_doe
Received data from: john_doe
```

```
[john_doe] Handling message with offset 40, type signature_reconstructed
Successfully processed message with offset 40, type signature_reconstructed
```

By performing the recover operation participants reconstructure the signature using the collected partial signs and share the reconstructured signatures between each other. These signatures get stored then and can be viewed at any time by running the following command that will show you a list of broadcasted reconstructed signatures for a given DKG round.
```
./dc4bc_cli get_signatures c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
Signing ID: ca800cac-2c13-4a14-8ca3-72c36112c5e4
                DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Participant: john_doe
                Reconstructed signature for the data: g4l9p8na4cTywMQluRtwR6S/KOxgCXSuC0VFhH5RywaiJ6i2yjWSIQcyqWiCkb00FXN+z67OfDSUTx8l7MFU1MJsJwRXPx9rGaFeOHhQi5aqHOH8ChTQftZSJXv5u/ck

                DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Participant: jane_doe
                Reconstructed signature for the data: g4l9p8na4cTywMQluRtwR6S/KOxgCXSuC0VFhH5RywaiJ6i2yjWSIQcyqWiCkb00FXN+z67OfDSUTx8l7MFU1MJsJwRXPx9rGaFeOHhQi5aqHOH8ChTQftZSJXv5u/ck
```

You can verify any signature by executing `verify_signature` command inside the airgapped prompt:
```
>>> verify_signature
> Enter the DKGRoundIdentifier: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
> Enter the BLS signature: g4l9p8na4cTywMQluRtwR6S/KOxgCXSuC0VFhH5RywaiJ6i2yjWSIQcyqWiCkb00FXN+z67OfDSUTx8l7MFU1MJsJwRXPx9rGaFeOHhQi5aqHOH8ChTQftZSJXv5u/ck
> Enter the message which was signed (base64): dGhlIG1lc3NhZ2UgdG8gc2lnbgo=
Signature is correct!
```

Now the ceremony is over. 

### Reinitialize DKG

If you've lost all your states, communication keys, but your mnemonic for private DKG key is safe, it is possible to reinitialize the whole DKG to recover DKG master key. Please refer to [this guide](https://github.com/lidofinance/dc4bc/blob/master/HowToReinit.md) in order to do that.