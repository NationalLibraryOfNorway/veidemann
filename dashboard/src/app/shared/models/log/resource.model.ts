import {ApiError} from '../commons/api-error.model';
import {create} from '@bufbuild/protobuf';
import {PageLog_Resource as ResourceProto, PageLog_ResourceSchema} from '../../../../api/log/v1/resources_pb';

export class Resource {
  uri: string;
  fromCache: boolean;
  renderable: boolean;
  resourceType: string;
  mimeType: string;
  statusCode: number;
  discoveryPath: string;
  warcId: string;
  referrer: string;
  error: ApiError;
  method: string;

  constructor({
                uri = '',
                fromCache = false,
                renderable = false,
                resourceType = '',
                mimeType = '',
                statusCode = 0,
                discoveryPath = '',
                warcId = '',
                referrer = '',
                error = new ApiError(),
                method = ''
              }: Partial<Resource> = {}) {
    this.uri = uri;
    this.fromCache = fromCache;
    this.renderable = renderable;
    this.resourceType = resourceType;
    this.mimeType = mimeType;
    this.statusCode = statusCode;
    this.discoveryPath = discoveryPath;
    this.warcId = warcId;
    this.referrer = referrer;
    this.error = error;
    this.method = method;
  }

  static fromProto(proto: ResourceProto): Resource {
    return new Resource({
      uri: proto.uri,
      fromCache: proto.fromCache,
      renderable: proto.renderable,
      resourceType: proto.resourceType,
      mimeType: proto.contentType,
      statusCode: proto.statusCode,
      discoveryPath: proto.discoveryPath,
      warcId: proto.warcId,
      referrer: proto.referrer,
      error: ApiError.fromProto(proto.error),
      method: proto.method
    });
  }

  static toProto(resource: Resource): ResourceProto {
    return create(PageLog_ResourceSchema, {
      uri: resource.uri,
      fromCache: resource.fromCache,
      renderable: resource.renderable,
      resourceType: resource.resourceType,
      contentType: resource.mimeType,
      statusCode: resource.statusCode,
      discoveryPath: resource.discoveryPath,
      warcId: resource.warcId,
      referrer: resource.referrer,
      method: resource.method,
    });
  }
}
