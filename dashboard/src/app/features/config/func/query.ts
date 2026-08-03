import {create} from '@bufbuild/protobuf';
import {ConfigObject, Kind} from '../../../shared/models';
import {FieldMask} from '../../../../api/commons/v1/resources_pb';
import {ListRequest, ListRequestSchema} from '../../../../api/config/v1/config_pb';

export function createListRequest(kind: Kind, queryTemplate?: Partial<ConfigObject>, queryMask?: FieldMask) {
  const listRequest = create(ListRequestSchema, {kind: kind.valueOf()});

  if (queryTemplate) {
    const configObject = new ConfigObject();
    Object.assign(configObject, queryTemplate);

    listRequest.queryTemplate = ConfigObject.toProto(configObject);
  }
  if (queryMask) {
    listRequest.queryMask = queryMask;
  }
  return listRequest;
}
