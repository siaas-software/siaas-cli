// Run only against disposable copies of the exact reviewed Personal source tags.
import assert from "node:assert/strict";
import { readFileSync, writeFileSync } from "node:fs";
import { LocalDatabase } from "./src/lib/local/database.ts";
import { LOCAL_USER_ID } from "./src/lib/runtime.ts";

const [mode, profile, expectedPath] = process.argv.slice(2);
assert(["seed", "verify"].includes(mode));
assert(profile && expectedPath);
const tables = ["workspaces", "projects", "project_templates", "actions", "assistants",
  "knowledge_sources", "documents", "document_chunks", "conversations", "messages"];
const snapshot = db => Object.fromEntries(tables.map(table => [
  table, db.rows(table).sort((a,b) => a.id.localeCompare(b.id)),
]));
const verifyIntegrity = db => {
  assert.equal(db.sql.prepare("PRAGMA user_version").get().user_version, 4);
  assert.equal(db.sql.prepare("PRAGMA integrity_check").get().integrity_check, "ok");
  assert.deepEqual(db.sql.prepare("PRAGMA foreign_key_check").all(), []);
};
const db = new LocalDatabase(profile);
try {
  if (mode === "seed") {
    const tenant_id = LOCAL_USER_ID;
    const p = db.put("projects", { tenant_id, name: "Upgrade fixture project" });
    const a = db.put("assistants", { tenant_id, project_id: p.id, name: "Upgrade assistant" });
    const d = db.put("documents", { tenant_id, assistant_id: a.id, title: "Existing document", kind: "text", raw_text: "Preserve this document" });
    const c = db.put("conversations", { tenant_id, assistant_id: a.id, title: "Existing conversation" });
    db.put("messages", { tenant_id, assistant_id: a.id, conversation_id: c.id, role: "user", content: "Preserve this message" });
    const file = tenant_id + "/" + a.id + "/migration-fixture.txt";
    db.saveFile(file, Buffer.from("Preserve this file"));
    verifyIntegrity(db);
    writeFileSync(expectedPath, JSON.stringify({ rows: snapshot(db), file, project: p.id, document: d.id }));
  } else {
    const expected = JSON.parse(readFileSync(expectedPath, "utf8"));
    verifyIntegrity(db);
    assert.deepEqual(snapshot(db), expected.rows, "existing rows changed during RC5 open");
    assert.equal(readFileSync(db.filePath(expected.file), "utf8"), "Preserve this file");
    assert.deepEqual(db.rows("project_runs"), []);
    const run = db.put("project_runs", { tenant_id: LOCAL_USER_ID, project_id: expected.project, status: "running" });
    assert.equal(db.get("project_runs", run.id).project_id, expected.project);
    assert.deepEqual(snapshot(db), expected.rows, "new table operation changed existing rows");
    verifyIntegrity(db);
  }
} finally {
  db.close();
}
console.log("Personal profile fixture " + mode + " PASS");
