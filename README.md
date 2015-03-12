pushd
====================
An open source distributed pubsub(publish/subscribe) server

	                  _         _ 
	  _ __  _   _ ___| |__   __| |
	 | '_ \| | | / __| '_ \ / _` |
	 | |_) | |_| \__ \ | | | (_| |
	 | .__/ \__,_|___/_| |_|\__,_|
	 |_|                                       

[![Build Status](https://travis-ci.org/nicholaskh/pushd.svg?branch=master)](https://travis-ci.org/nicholaskh/pushd)

### Install

*	No install, just use bin/pushd.linux or bin/pushd.mac

### HowToUse

*	Stand alone
1.	Run Server: bin/pushd.(linux|mac)
2.	Run Client(eg. telnet): telnet localhost 2222
	- sub channel1
3.	Run another Client: telnet localhost 2222
	- pub channel1 hello

	Then Client 1 will receive the message "hello"
	
*	Distributed env config
1.	Set the 'servers' section to all the peers in the cluster
	- note: the server in the 'servers' section must equal the 'tcp_listen_addr' option
2.	For the config of every peer, set the 'tcp_listen_addr' to the corresponding address

	Then the cluster will serve as one server

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


### Terms

*	Serv
*	Engine
*	s2s

### Highlights

*   Self manageable cluster
*	Use a distributed Client-Server architecture
		
		example.net <--------------> im.example.com
		     ^                                ^
		     |                                |
		     v                                v
		   romeo                            juliet
*   Highly usage of mem to improve latancy & throughput
*   Full realtime internal stats export via http
*   Smart metrics with low overhead
	
### Contribs

*   https://github.com/phunt/zktop
*   https://github.com/toddlipcon/gremlins
*   http://www.slideshare.net/renatko/couchbase-performance-benchmarking
*   https://issues.apache.org/jira/browse/THRIFT-826 TSocket: Could not write

### TODO

*   golang uses /proc/sys/net/core/somaxconn as listener backlog
    - increase it if you need over 128(default) simultaneous outstanding connections
*   hot reload config
*   bloom filter for overmuch channels
*	separate multiple applications
*	use "jid" of one client to get the destination server more efficient
