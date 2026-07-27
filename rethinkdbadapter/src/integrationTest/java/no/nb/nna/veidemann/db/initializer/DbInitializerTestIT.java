/*
 * Copyright 2018 National Library of Norway.
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

import com.rethinkdb.net.Result;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.commons.db.DbUpgradeException;
import no.nb.nna.veidemann.db.AbstractRethinkDbIntegrationTest;
import no.nb.nna.veidemann.db.RethinkDbConnection;
import no.nb.nna.veidemann.db.Tables;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static com.rethinkdb.RethinkDB.r;
import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

public class DbInitializerTestIT extends AbstractRethinkDbIntegrationTest {
    public static RethinkDbConnection conn;

    @BeforeEach
    public void init() throws DbException {
        DbService dbService = configureDbService();
        deleteDatabaseIfPresent();
        conn = ((RethinkDbInitializer) dbService.getDbInitializer()).getDbConnection();
    }

    @AfterEach
    public void shutdown() throws DbException {
        cleanupDbService();
    }

    @Test
    public void initialize() throws DbException {
        DbService.getInstance().getDbInitializer().initialize();

        String version = conn.exec(r.table(Tables.SYSTEM.name).get("db_version").g("db_version"));
        assertThat(version).isEqualTo(CreateNewDb.DB_VERSION);

        long configObjectCount = conn.exec(r.table(Tables.CONFIG.name).count());
        assertThat(configObjectCount).isGreaterThan(0);

        Map<String, Object> o = conn.exec(r.table(Tables.CONFIG.name)
                .group("kind")
                .count()
                .ungroup()
                .map(doc -> r.array(doc.g("group").coerceTo("string"), doc.g("reduction")))
                .coerceTo("object")
        );
        assertThat(o.get("politenessConfig")).isEqualTo(1L);
        assertThat(o.get("browserScript")).isEqualTo(5L);
        assertThat(o.get("crawlJob")).isEqualTo(4L);
        assertThat(o.get("browserConfig")).isEqualTo(1L);
        assertThat(o.get("crawlConfig")).isEqualTo(1L);
        assertThat(o.get("crawlScheduleConfig")).isEqualTo(3L);
        assertThat(o.get("crawlHostGroupConfig")).isEqualTo(1L);

        try (Result<Map<String, Object>> configObjects = conn.exec(r.table(Tables.CONFIG.name))) {
            assertThat(configObjects.iterator()).toIterable()
                    .hasSize(18)
                    .allSatisfy(configObject -> {
                        assertThat(configObject.get("apiVersion")).isEqualTo("v1");
                        assertThat(configObject).containsKey("kind");
                        assertThat(configObject).containsKey((String) configObject.get("kind"));
                        assertThat(configObject).containsKey("meta");
                        assertThat(castMap(configObject.get("meta"))).containsKey("name");
                    });
        }

        try (Result<Map<String, Object>> configObjects = conn.exec(r.table(Tables.CONFIG.name)
                .filter(r.hashMap("kind", "browserConfig")))) {
            assertThat(configObjects.iterator()).toIterable()
                    .hasSize(1)
                    .allSatisfy(configObject -> {
                        assertThat(configObject.get("apiVersion")).isEqualTo("v1");
                        assertThat(configObject.get("kind")).isEqualTo("browserConfig");
                        assertThat(configObject).containsKey("browserConfig");
                        assertThat(castMap(configObject.get("browserConfig"))).containsEntry("maxInactivityTimeMs", 2000L);
                    });
        }
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> castMap(Object value) {
        return (Map<String, Object>) value;
    }

    @Test
    public void repair() throws DbException {
        DbService.getInstance().getDbInitializer().initialize();

        String version = conn.exec(r.table(Tables.SYSTEM.name).get("db_version").g("db_version"));
        assertThat(version).isEqualTo(CreateNewDb.DB_VERSION);

        List<String> tables = conn.exec(r.tableList());
        assertExpectedTables(tables);

        List<String> indexes = conn.exec(r.table(Tables.CONFIG.name).indexList());
        assertThat(indexes).containsOnly("configRefs", "kind_label_key", "label", "label_value", "lastModified", "lastModifiedBy", "name");

        conn.exec(r.tableDrop(Tables.SEEDS.name));
        conn.exec(r.table(Tables.CONFIG.name).indexDrop("configRefs"));

        tables = conn.exec(r.tableList());
        assertThat(tables).containsOnly(Tables.CONFIG.name, Tables.CRAWL_ENTITIES.name,
                Tables.EXECUTIONS.name, Tables.JOB_EXECUTIONS.name, Tables.SYSTEM.name, Tables.URI_QUEUE.name);

        indexes = conn.exec(r.table(Tables.CONFIG.name).indexList());
        assertThat(indexes).containsOnly("kind_label_key", "label", "label_value", "lastModified", "lastModifiedBy", "name");

        DbService.getInstance().getDbInitializer().initialize();

        tables = conn.exec(r.tableList());
        assertExpectedTables(tables);

        indexes = conn.exec(r.table(Tables.CONFIG.name).indexList());
        assertThat(indexes).containsOnly("configRefs", "kind_label_key", "label", "label_value", "lastModified", "lastModifiedBy", "name");
    }

    @Test
    public void rejectUnsupportedVersion() throws DbException {
        DbService.getInstance().getDbInitializer().initialize();
        conn.exec(r.table(Tables.SYSTEM.name).get("db_version").update(r.hashMap("db_version", "1.13")));

        assertThatThrownBy(() -> DbService.getInstance().getDbInitializer().initialize())
                .isInstanceOf(DbUpgradeException.class)
                .hasMessageContaining("Unsupported database version '1.13'");
    }

    private void assertExpectedTables(List<String> tables) {
        assertThat(tables).containsOnly(Tables.CONFIG.name, Tables.CRAWL_ENTITIES.name, Tables.SEEDS.name,
                Tables.EXECUTIONS.name, Tables.JOB_EXECUTIONS.name, Tables.SYSTEM.name, Tables.URI_QUEUE.name);
    }
}
