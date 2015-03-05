pushd
====================
An open source distributed pubsub(publish/subscribe) server

	 ____  _  _  ____  _  _  ____ 
	(  _ \/ )( \/ ___)/ )( \(    \
	 ) __/) \/ (\___ \) __ ( ) D (
	(__)  \____/(____/\_)(_/(____/

[![Build Status](https://travis-ci.org/nicholaskh/pushd.svg?branch=master)](https://travis-ci.org/nicholaskh/pushd)


### Commands

*	gettoken [[username]]: get the login token for username
*	auth [[username]] [[token]]: auth the token of the username
*	sub [[channel]]: subscribe one channels
*	pub [[channel]] [[msg]]: publish msg to one channel
*	unsub [[channel]]: unsubscribe one channel

### Terms

*	Serv
*   Engine
*	s2s

### Highlights

*   Self manageable cluster
*	Use a distributed Client-Server architecture
*   Highly usage of mem to improve latancy & throughput
*   Full realtime internal stats export via http
*   Smart metrics with low overhead

====================
	  example.net <--------------> im.example.com
	     ^								   ^
	     |                                |
	     v                                v
	   romeo           					juliet
	
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
*	benchmark
*	separate multiple applications
*	use "jid" of one client to get the destination server more efficient
