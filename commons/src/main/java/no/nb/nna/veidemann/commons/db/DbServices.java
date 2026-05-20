package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.commons.settings.CommonSettings;

import java.util.Iterator;
import java.util.ServiceLoader;

public final class DbServices {
    private DbServices() {
    }

    public static DbService connect(CommonSettings settings) throws DbConnectionException {
        DbServiceSPI service = loadService();
        service.connect(settings);
        return new DbService(service);
    }

    public static DbService connectAndInstall(CommonSettings settings) throws DbConnectionException {
        return DbService.install(connect(settings));
    }

    public static DbService wrap(DbServiceSPI service) {
        return new DbService(service);
    }

    private static DbServiceSPI loadService() throws DbConnectionException {
        ServiceLoader<DbServiceSPI> dbServiceLoader = ServiceLoader.load(DbServiceSPI.class);
        Iterator<DbServiceSPI> services = dbServiceLoader.iterator();
        if (!services.hasNext()) {
            throw new DbConnectionException("No database adapter found");
        }
        DbServiceSPI service = services.next();
        if (services.hasNext()) {
            throw new DbConnectionException("More than one database adapter found");
        }
        return service;
    }
}