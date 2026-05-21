/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.db.initializer;

import com.rethinkdb.RethinkDB;
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbInitializer;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.commons.db.DbUpgradeException;
import no.nb.nna.veidemann.db.RethinkDbConnection;
import no.nb.nna.veidemann.db.Tables;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Objects;

/**
 *
 */
public class RethinkDbInitializer implements DbInitializer {

    private static final Logger LOG = LoggerFactory.getLogger(RethinkDbInitializer.class);

    static final RethinkDB r = RethinkDB.r;

    private final RethinkDbConnection conn;

    public RethinkDbInitializer(RethinkDbConnection conn) {
        this.conn = conn;
    }

    public void initialize() throws DbUpgradeException, DbQueryException, DbConnectionException {
        String dbName = requireDbName();

        if (!(boolean) conn.exec(r.dbList().contains(dbName))) {
            LOG.info("Creating database: " + dbName);
            new CreateNewDb(dbName, conn).run();
            LOG.info("Populating database with default data");
            new PopulateDbWithDefaultData().run();
            LOG.info("DB initialized");
            return;
        }

        String version = getCurrentDbVersion();
        if (!CreateNewDb.DB_VERSION.equals(version)) {
            throw new DbUpgradeException("Unsupported database version '" + version
                    + "'. Only '" + CreateNewDb.DB_VERSION + "' is supported");
        }

        LOG.info("Database found and is current version: {}", version);
        new CreateNewDb(dbName, conn).run();
        LOG.info("DB initialized");
    }

    @Override
    public void delete() throws DbException {
        try {
            conn.exec(r.dbDrop(requireDbName()));
        } catch (DbException e) {
            if (!e.getMessage().matches("Database .* does not exist.")) {
                throw e;
            }
        }
    }

    public RethinkDbConnection getDbConnection() {
        return conn;
    }

    private String getCurrentDbVersion() throws DbQueryException, DbConnectionException {
        return conn.exec(r.table(Tables.SYSTEM.name).get("db_version").g("db_version"));
    }

    private String requireDbName() {
        return Objects.requireNonNull(conn.getConnection().db(),
                "Configured RethinkDB connection is missing a default database");
    }

}
