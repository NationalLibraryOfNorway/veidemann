package no.nb.nna.veidemann.db;

import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.commons.settings.CommonSettings;

public abstract class AbstractRethinkDbIntegrationTest {
    protected DbService configureDbService() throws DbException {
        if (!DbService.isConfigured()) {
            DbService.configure(new CommonSettings()
                    .withDbHost(System.getProperty("db.host"))
                    .withDbPort(Integer.parseInt(System.getProperty("db.port")))
                    .withDbName(System.getProperty("db.name", "test"))
                    .withDbUser(System.getProperty("db.user", "admin"))
                    .withDbPassword(System.getProperty("db.password", "")));
        }
        return DbService.getInstance();
    }

    protected void deleteDatabaseIfPresent() throws DbException {
        if (!DbService.isConfigured()) {
            return;
        }

        try {
            DbService.getInstance().getDbInitializer().delete();
        } catch (DbException e) {
            if (!e.getMessage().matches("Database .* does not exist.")) {
                throw e;
            }
        }
    }

    protected void cleanupDbService() throws DbException {
        try {
            deleteDatabaseIfPresent();
        } finally {
            if (DbService.isConfigured()) {
                DbService.getInstance().close();
            }
        }
    }
}