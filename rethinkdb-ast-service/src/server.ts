import { serve } from "@hono/node-server";
import { Hono } from "hono";
import { logger } from "hono/logger";

import { jsQueryToJson } from "./app.js";

export const app = new Hono();

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

function startServer() {
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
}

if (require.main === module) {
  startServer();
}
