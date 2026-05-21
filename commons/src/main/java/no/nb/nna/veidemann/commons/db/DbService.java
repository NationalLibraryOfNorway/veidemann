package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.commons.settings.CommonSettings;

public class DbService implements AutoCloseable {
    private static DbService instance;

    private final DbServiceSPI service;

    DbService(DbServiceSPI service) {
        this.service = service;
    }

    /**
     * Get the singleton instance.
     *
     * @return the single RethinkDbConnection instance
     */
    public static synchronized DbService getInstance() {
        if (instance == null) {
            throw new IllegalStateException("Connection is not configured");
        }
        return instance;
    }

    /**
     * Configure the singleton RethinkDbConnection.
     * <p/>
     * This method must be called before any usage.
     *
     * @param settings a {@link CommonSettings} object with connection parameters
     */
    public static synchronized DbService configure(CommonSettings settings) throws DbConnectionException {
        return DbServices.connectAndInstall(settings);
    }

    /**
     * Copnfigure the singleton DbService with a preconfigured service provider.
     * <p>
     * This method is mostly useful for unit tests giving the possibility of mocking db.
     *
     * @param service
     * @return
     */
    public static synchronized DbService configure(DbServiceSPI service) {
        return install(DbServices.wrap(service));
    }

    public static synchronized DbService install(DbService service) {
        if (instance != null) {
            throw new IllegalStateException("Connection is already configured");
        }
        instance = service;
        return instance;
    }

    public static boolean isConfigured() {
        return instance != null;
    }

    public ConfigAdapter getConfigAdapter() {
        return service.getConfigAdapter();
    }

    public ExecutionsAdapter getExecutionsAdapter() {
        return service.getExecutionsAdapter();
    }

    public DbQueryAdapter getDbQueryAdapter() {
        return service.getDbQueryAdapter();
    }

    public FrontierAdapter getFrontierAdapter() {
        return service.getFrontierAdapter();
    }

    public DbInitializer getDbInitializer() {
        return service.getDbInitializer();
    }

    @Override
    public void close() {
        service.close();
        instance = null;
    }
}
