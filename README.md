pushd
====================
An open source distributed pubsub/IM server

	                  _         _ 
	  _ __  _   _ ___| |__   __| |
	 | '_ \| | | / __| '_ \ / _` |
	 | |_) | |_| \__ \ | | | (_| |
	 | .__/ \__,_|___/_| |_|\__,_|
	 |_|                                       

[![Build Status](https://travis-ci.org/nicholaskh/pushd.svg?branch=master)](https://travis-ci.org/nicholaskh/pushd)

### Install

*	Nothing to do, just use bin/pushd.linux or bin/pushd.mac

### HowToUse

*	Should open port 2222(optional) and 2223 in firewall, we use that as the tcp server and s2s gateway

*	Stand alone
	- Delete 'etc_servers' section or leave it empty
	- Run Server: bin/pushd.(linux|mac)
	- Run Client(eg. telnet): telnet localhost 2222
		+ sub channel1
	- Run another Client: telnet localhost 2222
		+ pub channel1 hello

	Then Client 1 will receive the message "hello"
	
*	Distributed env config
	- Set the 'etc_servers' section to the zk server addr
	- For the config of every peer, set the 'tcp_listen_addr' to the corresponding address

	Then the cluster will serve as one server
	
### Architecture
![Pushd Architecture](https://raw.githubusercontent.com/nicholaskh/pushd/master/doc/Architecture.png)

### Commands

*	gettoken [[username]]
	- get the login token for username
*	auth [[username]] [[token]]
	- auth the token of the username
*	sub [[channel]
	- subscribe one channels
*	pub [[channel]] [[msg]]
	- publish msg to one channel
*	unsub [[channel]]
	- unsubscribe one channel
*	his [[channel]] [[timestamp]]
	- fetch the history of the channel from timestamp


### Terms

*	Serv
*	Engine
*	s2s

### Highlights

*	Support upto 100W connections/server
*	Self manageable cluster
*	Use a distributed Client-Server architecture
		
		example.net <--------------> im.example.com
		     ^                                ^
		     |                                |
		     v                                v
		   romeo                            juliet
*	Highly usage of mem to improve latancy & throughput
*	Full realtime internal stats export via http
*	Smart metrics with low overhead
*	Http system status report
*	Use Mongodb as the message storage database
	
###	 Contribs

*	https://github.com/phunt/zktop
*	https://github.com/samuel/go-zookeeper

### TODO

*	golang uses /proc/sys/net/core/somaxconn as listener backlog
	- increase it if you need over 128(default) simultaneous outstanding connections
*  	hot reload config
*  	bloom filter for overmuch channels
*	separate multiple applications
*	for token stored in mongodb, add ttl index on expire column
	- db.token.ensureIndex( { "expire": 1 }, { expireAfterSeconds: 600 } )
