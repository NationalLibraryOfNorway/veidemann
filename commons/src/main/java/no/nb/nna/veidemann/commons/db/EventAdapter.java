package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.api.eventhandler.v1.DeleteResponse;
import no.nb.nna.veidemann.api.eventhandler.v1.EventObject;
import no.nb.nna.veidemann.api.eventhandler.v1.EventRef;
import no.nb.nna.veidemann.api.eventhandler.v1.ListCountResponse;
import no.nb.nna.veidemann.api.eventhandler.v1.ListLabelRequest;
import no.nb.nna.veidemann.api.eventhandler.v1.ListLabelResponse;
import no.nb.nna.veidemann.api.eventhandler.v1.SaveRequest;
import no.nb.nna.veidemann.api.eventhandler.v1.UpdateRequest;
import no.nb.nna.veidemann.api.eventhandler.v1.UpdateResponse;

public interface EventAdapter {

    EventObject getEventObject(EventRef request) throws DbException;

    ChangeFeed<EventObject> listEventObjects(no.nb.nna.veidemann.api.eventhandler.v1.ListRequest request) throws DbException;

    ListCountResponse countEventObjects(no.nb.nna.veidemann.api.eventhandler.v1.ListRequest request) throws DbException;

    EventObject saveEventObject(SaveRequest request) throws DbException;

    UpdateResponse updateEventObject(UpdateRequest request) throws DbException;

    DeleteResponse deleteEventObject(EventObject object) throws DbException;

    ListLabelResponse listLabels(ListLabelRequest request) throws DbException;
}
