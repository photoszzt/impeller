package kafka_consume_java;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.apache.kafka.common.errors.SerializationException;
import org.apache.kafka.common.serialization.Deserializer;
import org.apache.kafka.common.serialization.Serde;
import org.apache.kafka.common.serialization.Serializer;
import org.msgpack.jackson.dataformat.MessagePackMapper;

import java.io.IOException;
import java.util.Map;

public class MsgpPOJOSerde<T> implements Deserializer<T>, Serializer<T>, Serde<T> {
    private ObjectMapper objectMapper;
    private Class<T> cls;
    public MsgpPOJOSerde() {
        objectMapper = new MessagePackMapper();
        objectMapper.registerModule(new JavaTimeModule());
    }

    public void setClass(Class<T> cls) {
        this.cls = cls;
    }

    @Override
    public void configure(Map<String, ?> props, boolean isKey) {}

    @Override
    public void close() {}

    @Override
    public T deserialize(String s, byte[] bytes) {
        try {
            assert(bytes != null);
            T obj = (T)objectMapper.readValue(bytes, this.cls);
            assert(obj != null);
            return obj;
        } catch (IOException e) {
            e.printStackTrace();
            throw new SerializationException(e);
        }
    }

    @Override
    public Serializer<T> serializer() {
        return this;
    }

    @Override
    public Deserializer<T> deserializer() {
        return this;
    }

    @Override
    public byte[] serialize(String s, T t) {
        try {
            byte[] bytes = this.objectMapper.writeValueAsBytes(t);
            return bytes;
        } catch (JsonProcessingException e) {
            e.printStackTrace();
            throw new SerializationException(e);
        }
    }
}

