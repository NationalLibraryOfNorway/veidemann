const assert = require("node:assert/strict");
const test = require("node:test");

const { app } = require("../dist/server.js");

test("POST /ast serializes a ReQL query", async () => {
  const response = await app.request("/ast", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ query: 'r.table("foo").get("bar")' }),
  });

  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), {
    ast: '[16,[[15,["foo"]],"bar"]]',
  });
});
