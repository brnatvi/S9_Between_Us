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


racine + fils

racine
 => datum for each fils?


p -d :
directory => 
	1- file1, 
	2- folder1..etc
	0- exit  


Store peer info in local file

merkel tree -> abdou
udp transfers (getDatum/Datum) -> Natalia