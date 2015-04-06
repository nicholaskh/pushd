package pushd.client;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.net.Socket;
import java.net.InetSocketAddress;
import java.net.SocketTimeoutException;

public class PushdClient {

    private static final String MESSAGE_PREFIX_PUBLISHED = "PUBLISHED";
    private static final String MESSAGE_PREFIX_SUBSCRIBED = "SUBSCRIBED";
    private static final String MESSAGE_PREFIX_UNSUBSCRIBED = "UNSUBSCRIBED";

    private Socket sock;
    private PrintStream out;
    private BufferedReader in;
    private PushdProcessor processor;

    private Thread subThread;

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
    }

    private class ReadThread extends Thread {

        private PushdClient client;

        public ReadThread(PushdClient client) {
            this.client = client;
        }

        public void run() {
            while (true) {
                try {
                    char[] buf = new char[1000];
                    String str;
                    String msg;
                    long ts;
                    int lastSpace;
                    while (true) {
                        System.out.println("start read");
                        int num = this.client.in.read(buf, 0, 1000);
                        if (num == 0) {
                            System.out.println("eof");
                            this.client.close();
                            return;
                        }
                        str = String.valueOf(buf);
                        System.out.println("end read\n");
                        System.out.println(str);
                        if (str.startsWith("\0")) {
                            str = str.trim();
                            lastSpace = str.lastIndexOf(' ');
                            msg = str.substring(0, lastSpace);
                            ts = Long.parseLong(str.substring(lastSpace + 1));
                            this.client.processor.RecvMessage(msg, ts);
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

    public void close() throws IOException {
        this.in.close();
        this.out.close();
        this.sock.close();
    }

    public static void main(String[] args) throws Exception {
        PushdClient client = new PushdClient(new PushdProcessor(){

            public void RecvMessage(String msg, long ts) {
                System.out.printf("Time[%d] received message: %s\n", ts, msg);
            }

        });
        client.connect("127.0.0.1", 2222);
        System.out.println("start sub");
        client.subscribe("c1");
        System.out.println("end sub\n");
        client.subThread.join();
    }

    /*
    public static void main(String[] args) throws IOException {
        Socket client = new Socket("127.0.0.1", 20006);
        client.setSoTimeout(10000);
        BufferedReader input = new BufferedReader(new InputStreamReader(System.in));
        PrintStream out = new PrintStream(client.getOutputStream());
        BufferedReader buf =  new BufferedReader(new InputStreamReader(client.getInputStream()));
        boolean flag = true;
        while(flag){
            System.out.print("pls input: ");
            String str = input.readLine();
            out.println(str);
            if("bye".equals(str)){
                flag = false;
            }else{
                try{
                    String echo = buf.readLine();
                    System.out.println(echo);
                }catch(SocketTimeoutException e){
                    System.out.println("Time out, No response");
                }
            }
        }
        input.close();
        if(client != null){
            client.close();
        }
    }
    */

}
