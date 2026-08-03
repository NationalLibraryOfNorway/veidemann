import {create} from '@bufbuild/protobuf';
import {CrawlEntity as CrawlEntityProto, CrawlEntitySchema} from '../../../../api/config/v1/resources_pb';
import {ConfigObject} from './configobject.model';

export class CrawlEntity {

  static fromProto(proto: CrawlEntityProto): CrawlEntity {
    return new CrawlEntity();
  }

  static mergeConfigs(configObjects: ConfigObject[]): CrawlEntity {
    return new CrawlEntity();
  }

  static toProto(crawlEntity: CrawlEntity): CrawlEntityProto {
    return create(CrawlEntitySchema);
  }
}
