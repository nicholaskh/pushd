package pushd.client;

public interface PushdProcessor {
    void RecvMessage(String msg, long ts);
    void OnPingRecved(String pong);
    void OnSubscribeRecved(String cmd, String channel);
    void OnPublishRecved(String cmd);
    void OnUnsubscribeRecved(String cmd, String channel);
}
