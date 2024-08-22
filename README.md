## Overview:

The project was developed for the course "Protocoles des services Internet". It is a client application for the pear-to-pear network, running on the basis of a server hosted /when it becomes available/ at the address given below (see "ServerName").

The server works to establish primary contact between the peers, to overcome the NAT. Further communication should be carried out directly between the feasts.

## Usage:
### Run
```
go client.go ServerName MyPeerName Mode [... extra parameters]
```
**ServerName** = jch.irif.fr

**Mode** can have 3 values: `Client`, `Server`, `Menu`

For `Client` mode next operations are avalable:

###### `ServerInfo` - display on the screen list of the peers, address, keys, root
  
###### `PeerInfo` - display on the screen list of the peers, address, keys, root
  
> Example: `go client.go ServerName MyPeerName Client PeerInfo Peer`
>   
> Where ***Peer*** is a peer name	 
     
###### `HashesInfo` - display on the screen hashes and associated names
  
> Example: `go client.go ServerName MyPeerName Client HashesInfo Peer`
> 
> Where ***Peer*** is a peer name	 
     
###### `DownloadHash` - download data by hash
  
> Example: `go client.go ServerName MyPeerName Client DownloadHash Peer HASH DownloadDir`
> 
> Where ***Peer*** is a peer name,
> 
> Where ***HASH*** is 64 char string composed of hex literals,
> 
> Where ***DownloadDir*** is output directory on local HDD
	    
###### `DownloadPath` - download data by path
  
> Example: `go client.go ServerName MyPeerName Client DownloadPath Peer PATH DownloadDir`
> 
> Where ***Peer*** is a peer name
>             
> Where ***PATH*** is path on remote peer, for example /images/teachers.jpg
>   	    
> Where ***ownloadDir*** is output directory on local HDD


	    
For **Server** mode next operations are avalable:

  TODO
  
  
For **Menu** there is no extra parameters


### Examples:

 * go run client.go jch.irif.fr neon Client ServerInfo
   
 * go run client.go jch.irif.fr neon Client PeerInfo jch.irif.fr
   
 * go run client.go jch.irif.fr neon Client HashesInfo jch.irif.fr
   
 * go run client.go jch.irif.fr neon Client DownloadHash jch.irif.fr d31473e45414e71e1f900c97420afd301d59f3bf7b884c089382306dff281a30 ./Download/
   
 * go run client.go jch.irif.fr neon Client DownloadPath jch.irif.fr /images ./Download/


### The list of active peers can be seen here:
https://jch.irif.fr:8443
