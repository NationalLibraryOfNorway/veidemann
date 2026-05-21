# DB Initializer Maintenance

This package owns the current RethinkDB schema for `rethinkdbadapter`.

The model is intentionally simple:

- `CreateNewDb` defines the current schema and the current schema version.
- `RethinkDbInitializer` supports only two states:
  - the configured database does not exist yet: create it and populate default data
  - the configured database already exists and is already on the current version: repair missing tables or indexes idempotently
- Any other database version is rejected with `DbUpgradeException`.

There is no in-repo upgrade chain anymore. Do not add new `Upgrade*` classes.

## Current source of truth

- Schema version: `CreateNewDb.DB_VERSION`
- Schema creation and repair: `CreateNewDb`
- Initialization entry point: `RethinkDbInitializer`
- Default seed data: `PopulateDbWithDefaultData`
- Table and index helpers: `TableCreator`

## When to create a new version

Create a new version when the persisted database contract changes, for example:

- a table is added or removed
- an index is added, removed, or renamed
- required default data changes
- document shape changes in a way that must be reflected in a fresh database

Do not create a new version for pure code refactors that leave the persisted schema unchanged.

## How to make a schema change

1. Update the table inventory if needed.

If you add or remove a table, update `no.nb.nna.veidemann.db.Tables` first.

2. Change the current schema in `CreateNewDb`.

Edit `CreateNewDb` so it defines the new desired end state:

- add or remove table creation calls
- add or remove index creation calls
- keep the operations idempotent

`CreateNewDb` is used both for first-time creation and for repairing a database that is already on the current version, so it must be safe to run repeatedly.

3. Bump `CreateNewDb.DB_VERSION`.

Every schema change must also update `DB_VERSION`.

Example:

```java
public static final String DB_VERSION = "1.15";
```

4. Update default objects if the initial contents changed.

If the schema change affects the default configuration data, update:

- `PopulateDbWithDefaultData.java`
- files under `rethinkdbadapter/src/main/resources/default_objects/`

5. Update runtime code that depends on the schema.

Typical places:

- adapters in `no.nb.nna.veidemann.db`
- query builders or RethinkDB queries
- tests that assert tables, indexes, or seeded objects

## Important consequence of bumping the version

Because there is no automatic upgrade path, bumping `DB_VERSION` means an existing database with the previous version will fail initialization until it is handled outside this package.

You must decide one of these before shipping:

1. Recreate the database from scratch.
2. Perform a manual or operational migration outside the Java initializer.
3. Keep the schema version unchanged because the change was only repair-safe and should still be treated as the same schema version.

Do not silently widen `RethinkDbInitializer` to auto-upgrade old versions unless there is a deliberate decision to bring back managed migrations.

## Tests to update

At minimum, review and update:

- `rethinkdbadapter/src/integrationTest/java/no/nb/nna/veidemann/db/initializer/DbInitializerTestIT.java`
- `rethinkdbadapter/src/integrationTest/java/no/nb/nna/veidemann/db/RethinkDbConfigAdapterIT.java`
- `rethinkdbadapter/src/integrationTest/java/no/nb/nna/veidemann/db/RethinkDbExecutionsAdapterIT.java`

`DbInitializerTestIT` should verify the current table set, index set, seeded defaults, and unsupported-version behavior.

## Validation commands

Run the narrowest checks that prove the change.

Compile:

```bash
./gradlew :rethinkdbadapter:compileJava :rethinkdbadapter:compileIntegrationTestJava --console=plain
```

Unit tests:

```bash
./gradlew :rethinkdbadapter:test --console=plain
```

Integration tests against a live RethinkDB:

```bash
./gradlew :rethinkdbadapter:integrationTest -Ddb.host=localhost -Ddb.port=28015 --console=plain
```

Notes:

- the integration test task defaults `db.name` to `test`
- each integration test now drops the configured test database in setup and teardown

## Practical checklist

Before considering a schema version change done, verify all of this:

- `DB_VERSION` was bumped if the persisted contract changed
- `CreateNewDb` matches the new desired end state
- default objects were updated if needed
- affected queries and adapters were updated
- initializer integration tests assert the new table and index set
- integration tests pass against a live RethinkDB instance
