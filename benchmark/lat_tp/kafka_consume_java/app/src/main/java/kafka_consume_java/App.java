/*
 * This Java source file was generated by the Gradle 'init' task.
 */
package kafka_consume_java;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import org.apache.commons.cli.CommandLine;
import org.apache.commons.cli.DefaultParser;
import org.apache.commons.cli.HelpFormatter;
import org.apache.commons.cli.Option;
import org.apache.commons.cli.Options;
import org.apache.commons.cli.ParseException;
import org.apache.kafka.clients.consumer.ConsumerConfig;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.apache.kafka.clients.consumer.ConsumerRecords;
import org.apache.kafka.clients.consumer.KafkaConsumer;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.time.Duration;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Properties;
import java.util.concurrent.CountDownLatch;

public class App {
    private static final Option BOOTSTRAP_SERVER = new Option("b", "bootstrap-server", true, "Kafka bootstrap server");
    private static final Option EVENTS_NUM = new Option("ev", "events-num", true, "Number of events to produce");
    private static final Option DURATION = new Option("d", "duration", true, "Duration of the test in seconds");
    private static final Option WARMUP_EVENTS = new Option("we", "warmup-events", true, "Number of warmup events");
    private static final String TOPIC_NAME = "src";

    public static void main(String[] args) throws ParseException, IOException {
        Options options = getOptions();
        if (args == null || args.length == 0) {
            HelpFormatter formatter = new HelpFormatter();
            formatter.printHelp("java -jar kafka-consume-java.jar", options);
            System.exit(1);
        }
        DefaultParser parser = new DefaultParser();
        CommandLine line = parser.parse(options, args, true);
        String bootstrapServer = line.getOptionValue(BOOTSTRAP_SERVER.getOpt());
        String eventsNumStr = line.getOptionValue(EVENTS_NUM.getOpt());
        String durString = line.getOptionValue(DURATION.getOpt());
        String warmEventsString = line.getOptionValue(WARMUP_EVENTS.getOpt());
        int warmEvents = Integer.parseInt(warmEventsString);
        Duration dur = Duration.ofSeconds(Long.parseLong(durString));
        long durNano = dur.toNanos();
        long events = Long.parseLong(eventsNumStr);
        KafkaConsumer<String, byte[]> consumer = createKafkaConsumer(bootstrapServer);
        consumer.subscribe(Collections.singleton(TOPIC_NAME));
        long commitEveryNano = Duration.ofMillis(100).getNano();
        final CountDownLatch latch = new CountDownLatch(1);
        Runnable r = () -> {
            MsgpPOJOSerde<PayloadTs> payloadSerde = new MsgpPOJOSerde<>();
            payloadSerde.setClass(PayloadTs.class);
            int idx = 0;
            ArrayList<Long> prodToConLat = new ArrayList<>(2048);
            long rest = events;
            long consumedInWarmup = 0;
            if (warmEvents > 0) {
                System.out.println("Warming up...");
                consumedInWarmup = runLoop(warmEvents, consumer, commitEveryNano);
                rest = events - consumedInWarmup;
                System.out.println(warmEventsString + " warmup events consumed");
            }
            long startNano = System.nanoTime();
            long commitTimer = System.nanoTime();
            boolean hasUncommited = false;
            while (true) {
                if (System.nanoTime() - commitTimer >= commitEveryNano) {
                    consumer.commitSync();
                    commitTimer = System.nanoTime();
                    hasUncommited = false;
                }
                if ((durNano > 0 && System.nanoTime() - startNano > durNano) || (rest > 0 && idx >= rest)) {
                    break;
                }
                ConsumerRecords<String, byte[]> cr = consumer.poll(Duration.ofMillis(5));
                for (ConsumerRecord<String, byte[]> c : cr.records(TOPIC_NAME)) {
                    PayloadTs pts = payloadSerde.deserialize(TOPIC_NAME, c.value());
                    Instant ts = Instant.EPOCH.plus(pts.ts, ChronoUnit.MICROS);
                    long lat = Duration.between(ts, Instant.now()).toNanos() / 1000;
                    prodToConLat.add(lat);
                    idx += 1;
                    if (!hasUncommited) {
                        hasUncommited = true;
                    }
                }
            }
            if (hasUncommited) {
                consumer.commitSync();
            }
            Duration totalTime = Duration.of(System.nanoTime() - startNano, ChronoUnit.NANOS);
            System.out.println("\n" + prodToConLat + "\n");
            Collections.sort(prodToConLat);

            System.out.println("consumed " + (idx + consumedInWarmup) + " events, time: " + totalTime + ", throughput: " +
                    (double) idx / ((double) totalTime.toNanos() / 1_000_000_000) + ", p50: " +
                    P(prodToConLat, 0.5) + " us, p99: " + P(prodToConLat, 0.99) + " us");
            System.out.println("done consume");
            payloadSerde.close();
            latch.countDown();
        };
        Thread t = new Thread(r);
        HttpServer server = HttpServer.create(new InetSocketAddress(8090), 0);
        server.createContext("/consume", new HttpHandler() {
            @Override
            public void handle(HttpExchange exchange) throws IOException {
                System.out.println("Got connection\n");

                t.start();
                System.out.println("Start processing and waiting for result\n");
                try {
                    latch.await();
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
                byte[] response = "OK".getBytes();
                exchange.sendResponseHeaders(200, response.length);
                OutputStream os = exchange.getResponseBody();
                os.write(response);
                os.close();
            }
        });
        server.setExecutor(null);
        server.start();
    }

    private static int runLoop(int warmEvents,
            KafkaConsumer<String, byte[]> consumer, long commitEveryNano ) {
        int idx = 0;
        long commitTimer = System.nanoTime();
        boolean hasUncommited = false;
        while (idx < warmEvents) {
            if (hasUncommited && System.nanoTime() - commitTimer >= commitEveryNano) {
                consumer.commitSync();
                commitTimer = System.nanoTime();
                hasUncommited = false;
            }
            ConsumerRecords<String, byte[]> cr = consumer.poll(Duration.ofMillis(5));
            for (ConsumerRecord<String, byte[]> c : cr.records(TOPIC_NAME)) {
                idx += 1;
            }
        }
        if (hasUncommited) {
            consumer.commitSync();
        }
        return idx;
    }

    private static long P(ArrayList<Long> arr, double percent) {
        return arr.get((int) (((double) arr.size()) * percent + 0.5) - 1);
    }

    private static KafkaConsumer<String, byte[]> createKafkaConsumer(String bootstrapServer) {
        Properties props = new Properties();
        props.put(ConsumerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServer);
        props.put(ConsumerConfig.GROUP_ID_CONFIG, "bench");
        props.put(ConsumerConfig.AUTO_OFFSET_RESET_CONFIG, "earliest");
        props.put(ConsumerConfig.ENABLE_AUTO_COMMIT_CONFIG, "false");
        props.put(ConsumerConfig.KEY_DESERIALIZER_CLASS_CONFIG,
                "org.apache.kafka.common.serialization.StringDeserializer");
        props.put(ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG,
                "org.apache.kafka.common.serialization.ByteArrayDeserializer");
        return new KafkaConsumer<String, byte[]>(props);
    }

    private static Options getOptions() {
        Options options = new Options();
        options.addOption(BOOTSTRAP_SERVER);
        options.addOption(EVENTS_NUM);
        options.addOption(DURATION);
        options.addOption(WARMUP_EVENTS);
        return options;
    }
}
