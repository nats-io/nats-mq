package io.nats.mqbridge.examples;

import java.nio.charset.StandardCharsets;
import java.text.SimpleDateFormat;
import java.time.Duration;
import java.util.Calendar;
import java.util.Timer;
import java.util.TimerTask;

import io.nats.client.Connection;
import io.nats.client.NUID;
import io.nats.client.Nats;
import io.nats.client.Options;
import io.nats.mqbridge.Message;

public class Tick {
    public static void main(String args[]) {

        if (args.length < 2) {
            System.out.println("usage: Tick serverURL message");
            return;
        }

        try {
            Options o = new Options.Builder().connectionName("MQ-NATS Bridge tick-tock (ticker) example").server(args[0]).build();
            Connection nc = Nats.connect(o);
            byte[] body = args[1].getBytes(StandardCharsets.UTF_8);
            Timer timer = new Timer("tick");

            timer.scheduleAtFixedRate(new TimerTask() {
                long counter;
                
                public void run() {
                    String subject = "tick";
                    Message bridgeMessage = Message.NewMessageWithBody(body);

                    this.counter++;
                    String theTime = new SimpleDateFormat("E M dd HH:mm:ss z yyyy").format(Calendar.getInstance().getTime());
                    String corrID = NUID.nextGlobal();

                    bridgeMessage.setLongProperty("counter", this.counter);
                    bridgeMessage.setStringProperty("time", theTime);
                    bridgeMessage.getHeader().setCorrelID(corrID.getBytes(StandardCharsets.UTF_8));

                    try {
                        System.out.println("Sending message:");
			            System.out.printf("\tbody: %s\n", new String(bridgeMessage.getBody(), StandardCharsets.UTF_8));

                        long counter = bridgeMessage.getLongProperty("counter");
                        System.out.printf("\tcounter: %d\n", counter);

                        String time = bridgeMessage.getStringProperty("time");
                        System.out.printf("\ttime: %s\n", time);

                        System.out.printf("\tid: %s\n", new String(bridgeMessage.getHeader().getCorrelID(), StandardCharsets.UTF_8));
                        System.out.println();
            
                        byte[] encoded = bridgeMessage.encode();
                        nc.publish(subject, encoded);
                        nc.flush(Duration.ofSeconds(5));
                    } catch(Exception e) {
                        e.printStackTrace();
                        System.exit(-1);
                    }
                }
            }, 1000L, 1000L);
        } catch (Exception e) {
            e.printStackTrace();
            System.exit(-1);
        }
    }
}