package no.nb.nna.veidemann.frontier.db.script;

import java.io.IOException;
import java.io.InputStream;
import java.io.UncheckedIOException;
import java.nio.charset.StandardCharsets;
import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.Marker;
import org.slf4j.MarkerFactory;

import redis.clients.jedis.Jedis;
import redis.clients.jedis.exceptions.JedisNoScriptException;

public class LuaScript {
    private static final Logger LOG = LoggerFactory.getLogger(LuaScript.class);

    private final Marker scriptNameMarker;
    String scriptName;
    String sha;
    String script;

    public LuaScript(String scriptName) {
        scriptNameMarker = MarkerFactory.getMarker(scriptName);
        this.scriptName = scriptName;

        try (InputStream in = LuaScript.class.getClassLoader().getResourceAsStream("lua/" + scriptName)) {
            
            if (in == null) {
                throw new IllegalArgumentException("Lua script not found: " + scriptName);
            }
            script = new String(in.readAllBytes(), StandardCharsets.UTF_8);
        } catch (IOException e) {
            throw new UncheckedIOException("Failed loading Lua script: " + scriptName, e);
        }
    }

    Object runString(Jedis jedis, List<String> keys, List<String> args) {
        if (sha == null) {
            sha = jedis.scriptLoad(script);
        }
        try {
            Object result = jedis.evalsha(sha, keys, args);
            LOG.trace(scriptNameMarker, "{}: KEYS: {}, ARGS: {}, RESULT: {}", scriptName, keys, args, result);
            return result;
        } catch (JedisNoScriptException ex) {
            sha = null;
            return runString(jedis, keys, args);
        }
    }

    Object runBytes(Jedis jedis, List<byte[]> keys, List<byte[]> args) {
        LOG.trace(scriptNameMarker, "{}: KEYS: {}, ARGS: {}", scriptName, keys, args);
        if (sha == null) {
            sha = jedis.scriptLoad(script);
        }
        try {
            Object result = jedis.evalsha(sha.getBytes(), keys, args);
            LOG.trace(scriptNameMarker, "{}: KEYS: {}, ARGS: {}, RESULT: {}", scriptName, keys, args, result);
            return result;
        } catch (JedisNoScriptException ex) {
            sha = null;
            return runBytes(jedis, keys, args);
        }
    }
}
