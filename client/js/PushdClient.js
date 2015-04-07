//dependent on jquery

var PushdClient = {

    addr: "",
    subscribeTimeStamp: 0,

    onHistory: null,
    onPublish: null,
    onSubscribe: null,

    init: function(addr) {
        this.addr = addr;
        return this;
    },

    publish: function(channel, msg) {
        $.get('http://' + this.addr + '/pub/' + channel + '/' + msg, null, function(data) {
            if (PushdClient.onPublish) {
                PushdClient.onPublish(data);
            }
        });
    },
    
    subscribe: function(channel) {
        $.get('http://' + this.addr + '/sub/' + channel, null, function(data) {
            if (PushdClient.onSubscribe) {
                PushdClient.onSubscribe(data);
            }
        });
    },

    history: function(channel, ts) {
        $.getJSON('http://' + this.addr + '/history/' + channel + '/' + ts, null, function(json) {
            if (PushdClient.onHistory) {
                PushdClient.onHistory(json);
            }
        });
    }

}
