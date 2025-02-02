package kafka_consume_java;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Objects;

public class PayloadTs {
    public long ts;
    public byte[] payload;

    @JsonCreator
    public PayloadTs(@JsonProperty("ts") long ts,
                     @JsonProperty("payload") byte[] payload) {
        this.ts = ts;
        this.payload = payload;
    }

    @Override
    public String toString() {
        return "PaylodTs: {ts: " + this.ts + ", payload: " + payload + "}";
    }

    @Override
    public int hashCode() {
        return Objects.hash(ts, payload);
    }
}
