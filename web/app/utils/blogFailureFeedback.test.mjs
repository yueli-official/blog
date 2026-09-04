import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("resolves failures from stable codes without raw exception messages", async () => {
  const source = await readFile(new URL("./blogFailureFeedback.ts", import.meta.url), "utf8");
  assert.match(source, /resolveFailureFeedback\(error/u);
  assert.match(source, /"blog\.slug_taken": "这个链接地址已被使用。"/u);
  assert.doesNotMatch(source, /error\.message|data\?\.message/u);
});
