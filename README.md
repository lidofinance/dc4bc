Participants start with a pair of communication keys and aim to collectively produce a threshold BLS key pair. The three main components of the process are:

1. The storage for all public messages sent by participants, potentially a safe and unsophisticated key-value database (or any other DB that can be used alike, e.g. MongoDB). Messages sent to the storage are signed by communication keys.

2. A client for each of the participants that can send messages to the storage and talk to her airgapped machine (see below).

3. An airgapped machine for each of the participants that will handle all the threshold cryptography-related operations, i.e., the DKG messages and partial signatures. Сlients and their airgapped machines will communicate through QR codes: if the client, say, will need to form a justification for the DKG phase, she will generate a QR code that will be scanned by the airgapped machine that encodes the request for justification.

The expected workflow goes as follows:

1. For the participants to be able to collectively distribute profits, a threshold signature is required. We propose that the participants use a threshold BLS signature and a %source% DKG protocol to obtain that signature.

2. At the very beginning, the parties have to decide that they are ready to partitipate in DKG. This can happen through any suitable channel, which will not be specified here.

3. Each party sends a message to the storage that tells other participants that she is ready to participate in DKG, signed by her communication key.

4. Based on the messages from step 3, the parties decide on the parameters of DKG (i.e., the number of participants). Although the DKG protocol that we are going to use allows for a certain amount of the participants to misbehave, we want any missing or corrupt message to abort the DKG. This means that we can simplify the DKG to have only 4 steps:

SendPublicKeys,
SendCommits,
SendDeals,
ProcessDeals,
ProcessResponses.

5. Each DKG step will be performed in a uniform way. For example, to send a PublicKey message, the client will have to:
    * Form a request to the airgapped machine to generate a pair of PK, SK and return the PK;
    * Feed the request into the airgapped machine (scan the QR-code);
    * Get the airgapped machine response and scan it back;
    * Sign the message containing the DKG public key with client's communication key;
    * Send the message to the storage;
    * Read other participants' messages from the storage and form further requests to the airgapped machine using those repsonses.

6. During DKG, some of the messages have to be sent peer-to-peer (privately), and we don't want them to be broadcasted. Those messages will be sent to the public storage anyway, but will be encrypted using communication keys.

