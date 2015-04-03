package pushd.client;

public interface PushdProcessor {
    void RecvMessage(String msg, long ts);
}
