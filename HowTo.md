## Step-by-step guide

Generate keys for your node:
```
$ ./dc4bc_d gen_keys --username john_doe --key_store_dbdsn /tmp/dc4bc_john_doe_key_store
```
Start the node:
```
$ ./dc4bc_d start --username john_doe --key_store_dbdsn /tmp/dc4bc_john_doe_key_store --listen_addr localhost:8080 --state_dbdsn /tmp/dc4bc_john_doe_state --storage_dbdsn 94.130.57.249:9092
```
Start the airgapped machine:
```
# First print your node's address; you will be prompted to enter it.
$ ./dc4bc_cli get_address --listen_addr localhost:8080
e0d8083f8a2d18f310bfbdc9649a83664470f46053ab53c105a054b08f9eff85
$ ./dc4bc_airgapped /tmp/dc4bc_john_doe_airgapped_state
```
Print your address, communication public key and encryption public key and save it somewhere for later use:
``` 
$ ./dc4bc_cli get_address --listen_addr localhost:8080
e0d8083f8a2d18f310bfbdc9649a83664470f46053ab53c105a054b08f9eff85
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
# Inside the airgapped shell:
>>> show_dkg_pub_key
sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
```

Now you want to start the DKG procedure. Some participant (possibly you) will execute this command:
```
$ ./dc4bc_cli start_dkg 3 2 --listen_addr localhost:8080
```

This tells the node to send an InitDKG message that proposes to run DKG for 2 participants with `threshold=2`. You will be prompted to enter some required information about the suggested participants:
```
$ ./dc4bc_cli start_dkg 2 2 --listen_addr localhost:8080
Enter a necessary data for participant 0:
Enter address: e0d8083f8a2d18f310bfbdc9649a83664470f46053ab53c105a054b08f9eff85
Enter pubkey (base64): 4NgIP4otGPMQv73JZJqDZkRw9GBTq1PBBaBUsI+e/4U=
Enter DKGPubKey (base64): sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
Enter a necessary data for participant 1:
Enter address: 11c56cfa74e2e221444557811d43de3c39af927071f79728edcb0a8f4b1936ea
Enter pubkey (base64): EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
Enter DKGPubKey (base64): kJbOTZSwOKWYfg1KD/VxfRDEfk7kSgMzYiALJaLn2HJ08x5kIJWqkzFi/Z0B3ZEgBJROOybWPMVnQOpQ/DQwxYbxa6kgOPPBnY5WshX14vkgAtv+gE062rWLtFVBqZI+
```

The message will be consumed by your node:
```
[john_doe] starting to poll messages from append-only log...
[john_doe] Starting HTTP server on address: localhost:8080
[john_doe] Handling message with offset 0, type event_sig_proposal_init
[john_doe] message event_sig_proposal_init done successfully from e0d8083f8a2d18f310bfbdc9649a83664470f46053ab53c105a054b08f9eff85
[john_doe] Successfully processed message with offset 0, type event_sig_proposal_init
```

Now you have a pending operation in your operation pool. Get the list of pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Operation ID: 6d98f39d-1b24-49ce-8473-4f5d934ab2dc
Operation: {"ID":"6d98f39d-1b24-49ce-8473-4f5d934ab2dc","Type":"state_sig_proposal_await_participants_confirmations","Payload":"W3siUGFydGljaXBhbnRJZCI6MCwiQWRkciI6ImUwZDgwODNmOGEyZDE4ZjMxMGJmYmRjOTY0OWE4MzY2NDQ3MGY0NjA1M2FiNTNjMTA1YTA1NGIwOGY5ZWZmODVcbiIsIlRocmVzaG9sZCI6Mn0seyJQYXJ0aWNpcGFudElkIjoxLCJBZGRyIjoiMTFjNTZjZmE3NGUyZTIyMTQ0NDU1NzgxMWQ0M2RlM2MzOWFmOTI3MDcxZjc5NzI4ZWRjYjBhOGY0YjE5MzZlYVxuIiwiVGhyZXNob2xkIjoyfV0=","ResultMsgs":null,"CreatedAt":"2020-09-11T14:28:54.343122+03:00","DKGIdentifier":"191fb020fd30edd891b066f72e5a5e3a","To":"","Event":""}
-----------------------------------------------------
```

Copy the Operation ID and make the node produce a QR-code for it:
```
$ ./dc4bc_cli get_operation_qr 6d98f39d-1b24-49ce-8473-4f5d934ab2dc --listen_addr localhost:8080
List of paths to QR codes for operation 6d98f39d-1b24-49ce-8473-4f5d934ab2dc:
0) QR code: /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-0
1) QR code: /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-1
```

A single operation might be split into several QR-codes. Open the first QR code in your default image viewer and take a photo of it:
```
open /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-0
```

Now go to `dc4bc_airgapped` prompt and enter:

```
>>> read_qr
```

A new window will be opened showing what your laptop's camera sees. Place the photo of QR from the previous step in fron of the camera and wait for the airgapped machine to scan it. You will have to scan all operation QR codes that were produced by the node.  