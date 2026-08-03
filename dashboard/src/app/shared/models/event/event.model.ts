import {create} from '@bufbuild/protobuf';
import {
  Activity_Change as ChangeProto,
  DataSchema,
  EventObject as EventObjectProto,
  EventObjectSchema,
  EventObject_Severity as SeverityProto,
  EventObject_State as StateProto,
} from '../../../../api/eventhandler/v1/resources_pb';
import {fromTimestampProto, intersectString} from '../../func';

export enum State {
  NEW = 0,
  OPEN = 1,
  CLOSED = 2
}

export enum Severity {
  INFO = 0,
  WARN = 1,
  ERROR = 2
}

export enum ChangeType {
  CREATED = 0,
  VALUE_CHANGED = 1,
  ARRAY_ADD = 2,
  ARRAY_DEL = 3
}

const stateFromProto: Record<StateProto, State> = {
  [StateProto.NEW]: State.NEW,
  [StateProto.OPEN]: State.OPEN,
  [StateProto.CLOSED]: State.CLOSED,
};

const stateToProto: Record<State, StateProto> = {
  [State.NEW]: StateProto.NEW,
  [State.OPEN]: StateProto.OPEN,
  [State.CLOSED]: StateProto.CLOSED,
};

const severityFromProto: Record<SeverityProto, Severity> = {
  [SeverityProto.INFO]: Severity.INFO,
  [SeverityProto.WARN]: Severity.WARN,
  [SeverityProto.ERROR]: Severity.ERROR,
};

const severityToProto: Record<Severity, SeverityProto> = {
  [Severity.INFO]: SeverityProto.INFO,
  [Severity.WARN]: SeverityProto.WARN,
  [Severity.ERROR]: SeverityProto.ERROR,
};

export class Data {
  key?: string;
  value?: string;

  constructor({
                key = '',
                value = ''
              } = {}) {
    this.key = key;
    this.value = value;
  }
}

export class Change {
  type: ChangeType;
  field: string;
  oldVal: string;
  newVal: string;

  constructor({
                type = ChangeType.CREATED,
                field = '',
                oldVal = '',
                newVal = ''
              } = {}) {
    this.type = type;
    this.field = field;
    this.oldVal = oldVal;
    this.newVal = newVal;
  }

  static fromProto(proto: ChangeProto): Change {
    return new Change({
      type: proto.type as unknown as ChangeType,
      field: proto.field,
      oldVal: proto.oldVal,
      newVal: proto.newVal
    });
  }

}

export class Activity {
  modifiedBy?: string;
  modifiedTime?: string;
  description?: Change[];
  comment?: string;

  constructor({
                modifiedBy = '',
                modifiedTime = '',
                description = [],
                comment = '',
              } = {}) {
    this.modifiedBy = modifiedBy;
    this.modifiedTime = modifiedTime;
    this.description = description;
    this.comment = comment;
  }
}


export class EventObject {
  id?: string;
  type?: string;
  source?: string;
  state: State;
  assignee?: string;
  activityList: Activity[];
  dataList: Data[];
  severity: Severity;
  labelList?: string[];

  constructor(eventObject: Partial<EventObject> = {}) {
    this.id = eventObject.id || '';
    this.type = eventObject.type || '';
    this.source = eventObject.source || '';
    this.state = eventObject.state ?? State.NEW;
    this.assignee = eventObject.assignee || '';
    this.activityList = eventObject.activityList || [];
    this.dataList = eventObject.dataList || [];
    this.severity = eventObject.severity ?? Severity.INFO;
    this.labelList = eventObject.labelList || [];
  }

  static fromProto(proto: EventObjectProto): EventObject {
    return new EventObject({
      id: proto.id,
      type: proto.type,
      source: proto.source,
      state: stateFromProto[proto.state],
      assignee: proto.assignee,
      activityList: proto.activity.map(activity => new Activity({
        modifiedTime: fromTimestampProto(activity.modifiedTime),
        modifiedBy: activity.modifiedBy,
        description: activity.description.map(Change.fromProto),
        comment: activity.comment
      })),
      dataList: proto.data.map(data => new Data({key: data.key, value: data.value})),
      severity: severityFromProto[proto.severity],
      labelList: proto.label
    });
  }

  static toProto(eventObject: EventObject): EventObjectProto {
    return create(EventObjectSchema, {
      id: eventObject.id,
      assignee: eventObject.assignee,
      severity: severityToProto[eventObject.severity],
      state: stateToProto[eventObject.state],
      source: eventObject.source,
      type: eventObject.type,
      data: eventObject.dataList.map(data => create(DataSchema, {key: data.key, value: data.value})),
      label: eventObject.labelList,
    });
  }

  static mergeEvents(eventObjects: EventObject[]): EventObject {
    const eventObject = new EventObject();
    const compareObj: EventObject = eventObjects[0];

    const equalAssignee = eventObjects.every(event => event.assignee === compareObj.assignee);

    const equalSeverity = eventObjects.every(event => event.severity === compareObj.severity);

    eventObject.labelList = eventObjects.map(c => c.labelList).reduce(intersectString);

    if (equalAssignee) {
      eventObject.assignee = compareObj.assignee;
    } else {
      eventObject.assignee = null;
    }

    if (equalSeverity) {
      eventObject.severity = compareObj.severity;
    } else {
      eventObject.severity = null;
    }

    return eventObject;
  }
}
