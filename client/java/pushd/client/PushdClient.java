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

    public PushdClient(PushdProcessor processor) {
        this.processor = processor;
    }

    public void connect(String host, int port) throws InterruptedException, IOException  {
        this.sock = new Socket();
        this.sock.connect(new InetSocketAddress(host, port));
        this.out = new PrintStream(this.sock.getOutputStream());
        this.in = new BufferedReader(new InputStreamReader(this.sock.getInputStream()));
        Thread thread = new ReadThread(this.in, this.processor);
        thread.start();
        thread.join();
    }

    private class ReadThread extends Thread {

        private BufferedReader in;
        private PushdProcessor processor;

        public ReadThread(BufferedReader in, PushdProcessor processor) {
            this.in = in;
            this.processor = processor;
        }

        public void run() {
            try {
                while (true) {
                    String ret = this.in.readLine();
                    System.out.println(ret);
                    if (ret.startsWith("\0")) {
                        this.processor.RecvMessage(ret, 1111);
                    }
                }
            } catch (IOException ex) {
            }
        }

    }

    public void publish(String channel, String msg) throws IOException {
        this.out.printf("pub %s %s", channel, msg);
    }

    public void subscribe(String channel) throws IOException {
        this.out.printf("sub %s", channel);
    }

    public void close() throws IOException {
        this.in.close();
        this.out.close();
        this.sock.close();
    }

    public static void main(String[] args) throws Exception {
        PushdClient client = new PushdClient(new PushdProcessor(){

            public void RecvMessage(String msg, long ts) {
                System.out.printf("Time[%d] received message: %s", ts, msg);
            }

        });
        client.connect("127.0.0.1", 2222);
        client.subscribe("c1");
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
