package pushd.client;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.net.Socket;
import java.net.InetSocketAddress;
import java.net.SocketTimeoutException;
import java.util.Timer;
import java.util.TimerTask;

public class PushdClient {

    private static final String MESSAGE_PREFIX_PUBLISHED = "PUBLISHED";
    private static final String MESSAGE_PREFIX_SUBSCRIBED = "SUBSCRIBED";
    private static final String MESSAGE_PREFIX_UNSUBSCRIBED = "UNSUBSCRIBED";
    private static final String MESSAGE_PREFIX_PONG = "pong";
    private static final String OUTPUT_DELIMITER = "\2";

    private static final int PING_INTERVAL = 2000;

    private Socket sock;
    private PrintStream out;
    private BufferedReader in;
    private PushdProcessor processor;

    private Thread subThread;

    private Timer pingTimer = new Timer();

    public PushdClient(PushdProcessor processor) {
        this.processor = processor;
    }

    public void connect(String host, int port) throws InterruptedException, IOException  {
        this.sock = new Socket();
        this.sock.connect(new InetSocketAddress(host, port));
        this.out = new PrintStream(this.sock.getOutputStream(), false);
        this.in = new BufferedReader(new InputStreamReader(this.sock.getInputStream()));
        this.subThread = new ReadThread(this);
        this.subThread.start();

        this.pingTimer.schedule(new PingTimerTask(this), PushdClient.PING_INTERVAL);
    }

    private static class PingTimerTask extends TimerTask {

        private PushdClient client;

        public PingTimerTask(PushdClient client) {
            this.client = client;
        }

        public void run() {
            if (!this.client.sock.isClosed()) {
                client.pingTimer.schedule(new PingTimerTask(this.client), PushdClient.PING_INTERVAL);
                try {
                    this.client.ping();
                } catch (IOException ex) {}
            }
        }
    }

    private class ReadThread extends Thread {

        private PushdClient client;

        public ReadThread(PushdClient client) {
            this.client = client;
        }

        public void run() {
            while (true) {
                try {
                    String str;
                    String msg;
                    long ts;
                    int lastSpace;
                    while (true) {
                        //System.out.println("start read");
                        char[] buf = new char[1000];
                        int num = this.client.in.read(buf, 0, 1000);
                        if (num < 1) {
                            System.out.println("eof");
                            this.client.close();
                            this.client.pingTimer.cancel();
                            this.client.pingTimer.purge();
                            return;
                        }
                        buf[num - 1] = 0;
                        str = String.valueOf(buf);
                        //System.out.println("end read\n");
                        //System.out.println(str);
                        for (String msgEach : str.split(PushdClient.OUTPUT_DELIMITER)) {
                            if (msgEach.startsWith("\0")) {
                                msgEach = msgEach.trim();
                                lastSpace = msgEach.lastIndexOf(' ');
                                msg = msgEach.substring(0, lastSpace);
                                ts = Long.parseLong(msgEach.substring(lastSpace + 1));
                                this.client.processor.RecvMessage(msg, ts);
                            } else {
                                msgEach = msgEach.trim();
                                if (msgEach.equals(PushdClient.MESSAGE_PREFIX_PONG)) {
                                    this.client.processor.OnPingRecved(msgEach);
                                } else if(msgEach.startsWith(PushdClient.MESSAGE_PREFIX_SUBSCRIBED)) {
                                    String[] parts = msgEach.split(" ");
                                    this.client.processor.OnSubscribeRecved(parts[0], parts[1]);
                                } else if(msgEach.equals(PushdClient.MESSAGE_PREFIX_PUBLISHED)) {
                                    this.client.processor.OnPublishRecved(msgEach);
                                } else if(msgEach.startsWith(PushdClient.MESSAGE_PREFIX_UNSUBSCRIBED)) {
                                    String[] parts = msgEach.split(" ");
                                    this.client.processor.OnUnsubscribeRecved(parts[0], parts[1]);
                                }
                            }
                        }
                    }
                } catch (IOException ex) {}
            }
        }

    }

    public void publish(String channel, String msg) throws IOException {
        this.out.print(String.format("pub %s %s", channel, msg));
    }

    public void subscribe(String channel) throws IOException {
        this.out.print(String.format("sub %s", channel));
    }

    public void ping() throws IOException {
        this.out.print("ping");
    }

    public void close() throws IOException {
        this.in.close();
        this.out.close();
        this.sock.close();
    }

    public static void main(String[] args) throws Exception {
        PushdClient client = new PushdClient(new PushdProcessor(){

            public void RecvMessage(String msg, long ts) {
                System.out.printf("Time[%d] received message: %s%n", ts, msg);
            }

            public void OnPingRecved(String pong) {
                System.out.println(pong);
            }

            public void OnSubscribeRecved(String cmd, String channel) {
                System.out.printf("ggg %s %s%n", cmd, channel);
            }

            public void OnPublishRecved(String cmd) {
                System.out.printf("hhh %s%n", cmd);
            }

            public void OnUnsubscribeRecved(String cmd, String channel) {
                System.out.printf("jjj %s %s%n", cmd, channel);
            }
        });
        client.connect("192.168.9.91", 2222);
        System.out.println("start sub");
        client.subscribe("c1");
        System.out.println("end sub\n");

        client.publish("c1", "abc");

        client.subThread.join();
    }

}
