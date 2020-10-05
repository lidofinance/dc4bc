# dc4bc: distributed custody for the beacon chain

The goal of ths project is to make a simple, secure framework to generate and use threshold signatures for infrequent financial transactions over Ethereum 2.0 Beacon Chain (BLS on BLS12-381 curve). dc4bc only deals with key generation and the signature process with all the user-side logic offloaded to applications using dc4bc as a service or an API.

For a better key management, we presume that, when used in production, private encryption keys and threshold signature related secrets reside in an airgapped machine or an HSM. For a better auditablity and testability, network protocol logic is implemented as a set of finite state machines that change state deterministically in response to a a stream of outside events. 

The main and, for now, only network communication primitive we use is a shared bulletin board in form of an authenticated append-only log. Different implementations of that log could be a shared file (for local development or testing), a trusted network service (e.g. Amazon S3 bucket), a federated blockchain between protocol participants or a public blockchain. 

# How to test this code?

Run the command below to run unit-tests:

```
make test-short
```

# How to run this code?

Please refer to [this page](HowTo.md) for a complete guide for running the minimal application testnet.

# Repository description

* `./airgapped` The Airgapped machine source code. All encryption- and DKG-related code can be found in this package;
* `./client` The Client source code. The Client can poll messages from the message board. It also sets up a local http-server to process incoming requests (e.g., "please start a new DKG round");
* `./cmd` Command line interfaces for the Airgapped machine and the Client. All entry points to dc4bc apps can be found here;
* `./dkg` This package is more of a library for maintaining all active DKG instances and data;
* `./fsm` The FSM source code. The FSM decides when we are ready to move to the next step during DKG and signing;
* `./qr` A library for reading QR codes that encode pending Operations (which are used for communication between The Client, and the Airgapped machine); 
* `./storage` Two Bulletin Board implementations: File storage for local debugging and Kafka storage for real-world scenarios.

## Moving parts

### Participants 
N participants, having a hot (connected to the network) node and a cold (airgapped) node. Participants all have two pair of keys: auth keys and encryption keys. PubAuthKey_i, PrivAuthKey_i, PubEncKey_i, PrivEncKey_i respectively Participant_i. Each participant also have a secret seed used to generate DKG messages.

Auth keys are stored on the hot node, encryption keys and a seed are stored on a cold node.

### Conference call

It's presumed participants can use a separate secure communication channel (let's call it Conference Call) to establish initial parameters: the set of participants, their identities and public authentification keys, the nature and connection parameters of a bulletin board and so on.


### Bulletin Board

The core communication/storage primitive for dc4bc is a bulletin board - a simple authenticated append-only log that can be accesed by all the participants and allows posting authentificated messages and polling for posted messages. We need BB to have two functions:
- post(message, signature)
- getMessages(offset = 0)
  - returns a list of all messages posted after the first <offset> one

This allows us to establish communication primitives:

- broadcast(message) by Participant_i:
    post(message, signature(message, PrivAuthKey_i))
- private_message(message, Participant_j):
    encrypted_message = { "to" : Participant_j, "message": encrypt(message, PubEncKey_j)}
    broadcast(encrypted_message)

Bulletin board can be constructed using a trusted centralized service a-la github/amazon, using public blockchain, or using a consensus between participants to establish a private blockchain. Anyway, it should be abstracted away in the client and signer both and easily switchable.

Bulletin board is only available on a hot node.

### Secure Channel

There is a secure comminication channel between a hot node and a cold node between each participant. We expect it to be a dead simple QR-code based asynchronous messaging protocol, but it can be something more complicated eventually, e.g. USB connection to the HSM. It's got two primitive functions:
- h2c_message(message) - send a message from hot node to cold node, returns message hash
- await_c2h_reply(hash(message)) - wait for reply from cold node


## DKG Process

1. Using a Conference Call, participants establish: the set of participants, public keys for authentfication and encryption, the nature and connection parameters of a bulletin board, step timeouts, threshold number.
2. Any participant broadcasts a DKG Startup Message, that contains the set of participants, and public keys for authentfication and encryption. Hash of that message later is used as a unique id of a DKG (used in messages to differentiate between multiple parallel DKGs if needed).
3. All participants broadcast their agreement to participate in this particular DKG within the agreed upon step timeout.
4. When all participants agree, every participant asks a cold node to publish a commit:
   1. message_hash = h2c_message(<start DKG with DKG_hash xxx, number of participants X, threshold Y>)
   2. broadcast(await_c2h_reply(message_hash))
5. When all participants publish a commit, every participant:
   1. h2c_message(<all commits>)
   2. message_hash = h2c_message(<send deals>)
   3. deals = await_c2h_reply(message_hash)
   4. for participant in participants:
      1. direct_message(participant, deal[participant])
6. When a pariticipant has recieved all the deals:
   1. They reconstruct the public key from the deals and broadcast it
7. If everyone broadcasts the same reconstructed public key, DKG completed successfully

If at any point something goes wrong (timeout reached, the deal is invalid, public key is not recinstucted equally, some of participants complain using a Conference Call) the DKG is aborted.

## Signature process
1. Any paricipant broadcast a message to sign upon.
2. All other participants signal their willingness to sign by broadcasting agreemen to sign that message.
3. When enough (>= threshold) participants broadcasted an agreement, every participant:
   1. message_hash = h2c_message(<send a partial signature for message "message" for threshold public key "key">)
   2. broadcast(await_c2h_reply(message_hash))
4. When enough (>= threshold) participants broadcasted a partial signature, threshold signature is reconstructed.
5. Someone broadcasts a partial signature.

If not enough participants signal their willingness to sign within a timeout or signal their rejection to sign, signature process is aborted.

We organize logic in the hot node as a set of simple state machines that change state only by external trigger, such as CLI command, message from cold node, or a new message on Bulletin Board. That way it can be easily tested and audited.


### Overview

Participants start with a pair of communication keys and aim to collectively produce a threshold BLS key pair. The three main components of the process are:

1. The storage for all public messages sent by participants, potentially a safe and unsophisticated key-value database (or any other DB that can be used alike, e.g. MongoDB). Messages sent to the storage are signed by communication keys.

2. A client for each of the participants that can send messages to the storage and talk to her airgapped machine (see below).

3. An airgapped machine for each of the participants that will handle all the threshold cryptography-related operations, i.e., the DKG messages and partial signatures. Сlients and their airgapped machines will communicate through QR codes: if the client, say, will need to form a justification for the DKG phase, she will generate a QR code that will be scanned by the airgapped machine that encodes the request for justification.

### DKG

The expected DKG workflow goes as follows:

1. For the participants to be able to collectively distribute profits, a threshold signature is required. We propose that the participants use a threshold BLS signature and a %source% DKG protocol to obtain that signature.

2. At the very beginning, the parties have to decide that they are ready to partitipate in DKG. This can happen through any suitable channel, which will not be specified here.

3. Each party sends a message to the storage that tells other participants that she is ready to participate in DKG, signed by her communication key.

4. Based on the messages from step 3, the parties decide on the parameters of DKG (i.e., the number of participants). Although the DKG protocol that we are going to use allows for a certain amount of the participants to misbehave, we want any missing or corrupt message to abort the DKG. This means that we can simplify the DKG to have only 4 steps:

    * SendPublicKeys,
    * SendCommits,
    * SendDeals,
    * ProcessDeals,
    * ProcessResponses.

5. Each DKG step will be performed in a uniform way. For example, to send a PublicKey message, the client will have to:
    * Form a request to the airgapped machine to generate a pair of PK, SK and return the PK;
    * Feed the request into the airgapped machine (scan the QR-code);
    * Get the airgapped machine response and scan it back;
    * Sign the message containing the DKG public key with client's communication key;
    * Send the message to the storage;
    * Read other participants' messages from the storage and form further requests to the airgapped machine using those repsonses.

6. During DKG, some of the messages have to be sent peer-to-peer (privately), and we don't want them to be broadcasted. Those messages will be sent to the public storage anyway, but will be encrypted using communication keys.

7. After the participants successfully finish DKG, their shares of BLS private key are stored safely on the airgapped machine.

8. When the participants decide to distribute profits, they get their partial signature from the airgapped machine and send it to the storage; after the required number of partial signatures is supplied, the collective signarute can be recovered.
