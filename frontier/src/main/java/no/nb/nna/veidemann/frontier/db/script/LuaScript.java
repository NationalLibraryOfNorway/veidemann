package no.nb.nna.veidemann.frontier.db.script;

import java.io.IOException;
import java.io.InputStream;
import java.io.UncheckedIOException;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Objects;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.Marker;
import org.slf4j.MarkerFactory;

import redis.clients.jedis.Jedis;
import redis.clients.jedis.exceptions.JedisNoScriptException;

public class LuaScript {
    private static final Logger LOG = LoggerFactory.getLogger(LuaScript.class);

    private final Marker scriptNameMarker;
    private final String scriptName;
    private final String script;
    private volatile String sha;

    public LuaScript(String scriptName) {
        this.scriptName = Objects.requireNonNull(scriptName, "scriptName");
        this.scriptNameMarker = MarkerFactory.getMarker(scriptName);

        try (InputStream in = LuaScript.class.getClassLoader().getResourceAsStream("lua/" + scriptName)) {
            if (in == null) {
                throw new IllegalArgumentException("Lua script not found: " + scriptName);
            }
            this.script = new String(in.readAllBytes(), StandardCharsets.UTF_8);
        } catch (IOException e) {
            throw new UncheckedIOException("Failed loading Lua script: " + scriptName, e);
        }
    }

    Object runString(Jedis jedis, List<String> keys, List<String> args) {
        return evalWithRetry(jedis, keys, args, false);
    }

    Object runBytes(Jedis jedis, List<byte[]> keys, List<byte[]> args) {
        return evalWithRetry(jedis, keys, args, true);
    }

    @SuppressWarnings("unchecked")
    private Object evalWithRetry(Jedis jedis, List<?> keys, List<?> args, boolean binary) {
        if (sha == null) {
            sha = jedis.scriptLoad(script);
        }
        try {
            Object result;
            if (binary) {
                result = jedis.evalsha(
                        sha.getBytes(StandardCharsets.UTF_8),
                        (List<byte[]>) keys,
                        (List<byte[]>) args);
            } else {
                result = jedis.evalsha(
                        sha,
                        (List<String>) keys,
                        (List<String>) args);
            }
            LOG.trace(scriptNameMarker, "{}: KEYS: {}, ARGS: {}, RESULT: {}", scriptName, keys, args, result);
            return result;
        } catch (JedisNoScriptException ex) {
            // Script cache flushed – reload and retry once
            sha = jedis.scriptLoad(script);
            Object result;
            if (binary) {
                result = jedis.evalsha(
                        sha.getBytes(StandardCharsets.UTF_8),
                        (List<byte[]>) keys,
                        (List<byte[]>) args);
            } else {
                result = jedis.evalsha(
                        sha,
                        (List<String>) keys,
                        (List<String>) args);
            }
            LOG.trace(scriptNameMarker, "{}(reload): KEYS: {}, ARGS: {}, RESULT: {}", scriptName, keys, args, result);
            return result;
        }
    }
}
