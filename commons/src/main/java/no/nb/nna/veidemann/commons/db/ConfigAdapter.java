package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.DeleteResponse;
import no.nb.nna.veidemann.api.config.v1.GetLabelKeysRequest;
import no.nb.nna.veidemann.api.config.v1.LabelKeysResponse;
import no.nb.nna.veidemann.api.config.v1.ListCountResponse;
import no.nb.nna.veidemann.api.config.v1.LogLevels;
import no.nb.nna.veidemann.api.config.v1.UpdateRequest;
import no.nb.nna.veidemann.api.config.v1.UpdateResponse;

public interface ConfigAdapter {
    ConfigObject getConfigObject(ConfigRef request) throws DbException;

    ChangeFeed<ConfigObject> listConfigObjects(no.nb.nna.veidemann.api.config.v1.ListRequest request) throws DbException;

    ListCountResponse countConfigObjects(no.nb.nna.veidemann.api.config.v1.ListRequest request) throws DbException;

    ConfigObject saveConfigObject(ConfigObject object) throws DbException;

    UpdateResponse updateConfigObjects(UpdateRequest request) throws DbException;

    DeleteResponse deleteConfigObject(ConfigObject object) throws DbException;

    LabelKeysResponse getLabelKeys(GetLabelKeysRequest request) throws DbException;

    LogLevels getLogConfig() throws DbException;

    LogLevels saveLogConfig(LogLevels logLevels) throws DbException;
}
