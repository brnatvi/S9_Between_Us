## L3 stuff : flots (stream)
kinda trivial info, he just talks about different stream and message oriented protocoles (**TCP** => stream | **UDP/WebSocket/sctp** => message)

## Project
### Data: 
 We're gonna use hashes to ensure the integrity of files.

 If file size is less than **1024 bytes**, its merkel tree is the hash itself, otherwise its split up to multiple parts (each less than **1024 bytes**), these parts (chunks) are represented as in tree from and every node's hash is just is its children's hashes concatenated, this way if a leaf is modified there are minimal changes to the tree (quite the elegant solution tbh).

 In the project we have 3 types of nodes:

 - chunk nodes: (defined in previous paragraph).
 - big file nodes: (defines in previous paragraph, the split up files), can have anywhere from 2-32 children.
 - directory nodes: represent a directory, can have anywhere from 0-16 children.

#### Body structure:
 - first byte:
    - 0 if its a chunk
    - 1 if its a big file
    - 2 if its a directory, followed by a series of entries Name(32 bytes)/Hash(32 bytes) representing its contents

### Sign up:

1. Client sends a **Hello** message to the server to sign up as a peer (Type = 2)
2. Server replies witha  **HelloReply** message (Type = 129)
3. Server sends a **PublicKey** message (Type = 3), to which the client must reply with a **PublicKeyReply** message (Type = 130) to confirm the sign up.
4. The sign up expires after 180s of inactivity

### REST (peer info):
 Perfectly and briefly explained in Section 4 of [project.pdf](https://www.irif.fr/~jch/enseignement/internet/projet.pdf)

### Data transfer:

**ALWAYS CHECK HASH VALUES!!** 

- p1 sends **GetDatum** message (Type = 5) asking for data sending its hash
- p2 replies with **Datum** message (Type = 130) + data with its hash, or **NoDatum** (Type = 133) containing the same hash that was received


### Error messages:
 - **Error** message (Type = 1), sends an error message with the details in **Body**, must be human readable
 - **ErrorReply** message (Type = 128), error when replying to a req, same syntax as **Error**  


### Submitting the project:
 lname1-lname2.tar.gz

->  extracted to directory lname1-lname2

 no compiling, just `make`

 email subject: Internet M2: lname1-lname2



### Traversing NAT:
(can ignore up until section 4.2.3 Nat)
1. **IP header** <- router checks this and modifies some fields:
    - hop count/ttl
    - checksum
    - QoS/ToS
2. **TCP/UDP header:** routers shouldn't open this but they do anyway
    - Queueing:
        - fair queueing (alternate whose packets to send)
    - Middlebox (EVIL!!) modifies stuff (au dessus de IP) without permission >:(
        1. Firewalls:
            - With state: ie. doesnt allow incoming traffic
    - Not a Middlebox:
        - IDS (Intrusion Detection System): a blackbox that uses heuristics to detect possible attacks and adds the source to the ACL
3. **Accelerators:**
    - fake replies from the network to keep the connection up (used in situations with very high latency ie. geostationary sattelites)
4. **NAT:**
    - because almost all devices exist behind a NAT:
        1. addresses arent the sole identifying information for a given device :'(
        2. we have state within the network :terrified:
            - wiped after rebooting :))
            - as the entries have an expiration date, need to resend a keepalive periodically
    - NAT traversal techniques:
        1. Client - Server : just need periodic keepalives and it works just fine
        2. P2P: to establish a  p2p conn through NATs we can either 
            1. direct conn (usually fails)
            2. send requests simultaneously
            3. use a STUN server:
                - both clients send a UDP request to the server
                - server replies with their external socket 
            - In the project we'll use a simplified STUN, as the server already has our addr, so we (kinda) skip the STUN step and jump straight to the sync part!z
            
        