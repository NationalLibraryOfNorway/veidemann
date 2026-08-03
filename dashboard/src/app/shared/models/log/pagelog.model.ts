import {Resource} from './resource.model';
import {create} from '@bufbuild/protobuf';
import {PageLog as PageLogProto, PageLogSchema} from '../../../../api/log/v1/resources_pb';

export class PageLog {
  id: string;
  warcId: string;
  uri: string;
  executionId: string;
  referrer: string;
  jobExecutionId: string;
  collectionFinalName: string;
  method: string;
  resource: Resource[];
  outlink: string[];

  constructor({
                id = '',
                warcId = '',
                uri = '',
                executionId = '',
                referrer = '',
                jobExecutionId = '',
                collectionFinalName = '',
                method = '',
                resource = [],
                outlink = [],
              }: Partial<PageLog> = {}) {
    this.id = id || warcId;
    this.warcId = warcId;
    this.uri = uri;
    this.executionId = executionId;
    this.referrer = referrer;
    this.jobExecutionId = jobExecutionId;
    this.collectionFinalName = collectionFinalName;
    this.method = method;
    this.resource = resource ? resource.map(_ => new Resource(_)) : [];
    this.outlink = outlink;
  }

  /**
   * A function that transforms the results. This function is called for each member of the object.
   * If a member contains nested objects, the nested objects are transformed before the parent object is.
   * @see JSON.parse
   */
  static reviver(key: string, value: any) {
    switch (key) {
      default:
        return value;
    }
  }

  static fromProto(proto: PageLogProto): PageLog {
    return new PageLog({
      warcId: proto.warcId,
      uri: proto.uri,
      executionId: proto.executionId,
      referrer: proto.referrer,
      jobExecutionId: proto.jobExecutionId,
      collectionFinalName: proto.collectionFinalName,
      method: proto.method,
      resource: proto.resource.map(Resource.fromProto),
      outlink: proto.outlink
    });
  }

  static toProto(pageLog: PageLog): PageLogProto {
    return create(PageLogSchema, {
      warcId: pageLog.warcId,
      uri: pageLog.uri,
      executionId: pageLog.executionId,
      referrer: pageLog.referrer,
      jobExecutionId: pageLog.jobExecutionId,
      collectionFinalName: pageLog.collectionFinalName,
      method: pageLog.method,
      resource: pageLog.resource.map(Resource.toProto),
      outlink: pageLog.outlink,
    });
  }
}
