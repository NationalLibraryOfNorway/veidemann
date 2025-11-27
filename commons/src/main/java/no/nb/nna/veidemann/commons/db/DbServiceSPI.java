package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.commons.settings.CommonSettings;

public interface DbServiceSPI extends AutoCloseable {
    void connect(CommonSettings settings) throws DbConnectionException;

    @Override
    void close();

    ConfigAdapter getConfigAdapter();

    ExecutionsAdapter getExecutionsAdapter();

    EventAdapter getEventAdapter();

    DbInitializer getDbInitializer();
}
