import { Hono } from "hono";
import { logger } from "hono/logger";
import { serve } from "@hono/node-server";
import { r } from "rethinkdb-ts";

function assertSafeReql(query: string) {
  const src = query.trim();

  if (!src.startsWith("r.")) {
    throw new Error('Query must start with "r."');
  }

  // Block obvious escape hatches / Node integration.
  // This is *not* a full sandbox, just a coarse filter.
  const forbidden = /\b(require|process|globalThis|global|module|Function|eval|import|child_process|fs|net|tls|http|https)\b/;

  if (forbidden.test(src)) {
    throw new Error("Query contains forbidden identifiers");
  }
}

/**
 * Turn a JS ReQL expression string into the serialized ReQL JSON
 * using rethinkdb-ts's r.serialize(term).
 *
 * Example input:  r.table("foo").get("bar")
 */
function jsQueryToJson(querySource: string): string {
  if (!querySource || typeof querySource !== "string") {
    throw new Error("query must be a non-empty string");
  }

  assertSafeReql(querySource);

  const fn = new Function("r", `return (${querySource});`);
  const term = fn(r);

  if (!term) {
    throw new Error("query did not return a value");
  }

  const serialized = (r as any).serialize(term);
  if (serialized == null) {
    throw new Error("r.serialize(term) returned null/undefined");
  }

  return typeof serialized === "string"
    ? serialized
    : JSON.stringify(serialized);
}

const app = new Hono();

app.use(logger());

/**
 * POST /ast
 *
 * Request JSON:
 *   { "query": "r.table(\"foo\").get(\"bar\")" }
 *
 * Response JSON:
 *   { "ast": "<reql-json-string>" }
 */
app.post("/ast", async (c) => {
    let body: any;
    try {
        body = await c.req.json();
    } catch {
        console.error("Invalid JSON body");
        return c.json({ error: "Invalid JSON" }, 400);
    }

    const query = String(body?.query ?? "");

    try {
        const astJson = jsQueryToJson(query);
        return c.json({ ast: astJson });
    } catch (err: any) {
        console.error("AST error for query:", query, "\n  ->", err);
        const message =
            typeof err?.message === "string"
                ? err.message
                : "Internal server error";
        return c.json({ error: message }, 400);
    }
});

/**
 * Simple health check
 */
app.get("/health", (c) => {
    return c.json({ status: "ok" });
});

const port = Number(process.env.PORT) || 3000;

const server = serve({ fetch: app.fetch, port });

function shutdown(signal: string) {
    console.log(`${signal} received, shutting down gracefully...`);
    server.close((err?: Error) => {
        if (err) {
            console.error(err);
            process.exit(1);
        }
        process.exit(0);
    });
}

process.on("SIGTERM", () => shutdown("SIGTERM"));
process.on("SIGINT", () => shutdown("SIGINT"));

console.log(`rethink-ast-service listening on http://localhost:${port}`);
