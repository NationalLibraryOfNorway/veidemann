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
export function jsQueryToJson(querySource: string): string {
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
