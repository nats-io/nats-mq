package io.nats.mqbridge.examples;

import java.nio.charset.StandardCharsets;
import java.time.Duration;

import io.nats.client.Connection;
import io.nats.client.Nats;
import io.nats.client.Options;
import io.nats.client.Subscription;
import io.nats.mqbridge.Message;

public class Tock {
    public static void main(String args[]) {

        if (args.length < 1) {
            System.out.println("usage: Tock serverURL");
            return;
        }

        try {
            Options o = new Options.Builder().connectionName("MQ-NATS Bridge tick-tock (ticker) example").server(args[0]).build();
            Connection nc = Nats.connect(o);
            Subscription sub = nc.subscribe("tock");

            System.out.println("Listening on tock...");

            while (true) {
                io.nats.client.Message msg = sub.nextMessage(Duration.ofSeconds(10));

                if (msg == null) {
                    continue;
                }

                Message bridgeMessage = Message.DecodeMessage(msg.getData());

                System.out.println("Received message:");
                System.out.printf("\tbody: %s\n", new String(bridgeMessage.getBody(), StandardCharsets.UTF_8));

                long counter = bridgeMessage.getLongProperty("counter");
                System.out.printf("\tcounter: %d\n", counter);

                String time = bridgeMessage.getStringProperty("time");
                System.out.printf("\ttime: %s\n", time);

                System.out.printf("\tid: %s\n", new String(bridgeMessage.getHeader().getCorrelID(), StandardCharsets.UTF_8));
                System.out.println();
            }
        } catch (Exception e) {
            e.printStackTrace();
            System.exit(-1);
        }
    }
}