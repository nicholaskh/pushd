//dependent on jquery

var PushdClient = {

    addr: "",
    subscribeTs: 0,

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
        $.get('http://' + this.addr + '/sub/' + channel + '/' + PushdClient.subscribeTs, null, function(data) {
            if (!data) {
                return;
            }
            if (data.indexOf('[') === 0) {
                json = eval(data);
                for (i in json) {
                    if (json[i]['ts'] > PushdClient.subscribeTs) {
                        PushdClient.subscribeTs = json[i]['ts'].toString();
                    }
                }
            } else {
                data = data.trim();
                var ts = data.substr(data.lastIndexOf(" ") + 1);
                json = [{
                    channel: channel,
                    msg: data.substr(0, data.lastIndexOf(" ")),
                    ts: ts,
                }];
                if (ts > PushdClient.subscribeTs) {
                    PushdClient.subscribeTs = ts;
                }
            }
            if (PushdClient.onSubscribe) {
                PushdClient.onSubscribe(json);
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
