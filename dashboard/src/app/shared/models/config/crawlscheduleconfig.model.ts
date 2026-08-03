import {create} from '@bufbuild/protobuf';
import {
  CrawlScheduleConfig as CrawlScheduleConfigProto,
  CrawlScheduleConfigSchema,
} from '../../../../api/config/v1/resources_pb';
import {fromTimestampProto, toTimestampProto} from '../../func';
import {ConfigObject} from './configobject.model';

export class CrawlScheduleConfig {
  cronExpression: string;
  validFrom?: string;
  validTo?: string;

  constructor({
                cronExpression = '',
                validFrom = '',
                validTo = '',
              }: Partial<CrawlScheduleConfig> = {}) {
    this.cronExpression = cronExpression;
    this.validFrom = validFrom;
    this.validTo = validTo;
  }

  static fromProto(proto: CrawlScheduleConfigProto): CrawlScheduleConfig {
    return new CrawlScheduleConfig({
      cronExpression: proto.cronExpression,
      validFrom: fromTimestampProto(proto.validFrom),
      validTo: fromTimestampProto(proto.validTo)
    });
  }

  static mergeConfigs(configObjects: ConfigObject[]): CrawlScheduleConfig {
    const crawlScheduleConfig = new CrawlScheduleConfig();
    const compareObj: CrawlScheduleConfig = configObjects[0].crawlScheduleConfig;

    const equalValidFrom = this.commonValidFrom(configObjects);

    const equalValidTo = this.commonValidTo(configObjects);

    if (equalValidFrom) {
      crawlScheduleConfig.validFrom = compareObj.validFrom;
    } else {
      crawlScheduleConfig.validFrom = undefined;
    }

    if (equalValidTo) {
      crawlScheduleConfig.validTo = compareObj.validTo;
    } else {
      crawlScheduleConfig.validTo = undefined;
    }
    return crawlScheduleConfig;
  }

  static commonValidFrom(configObjects: ConfigObject[]): boolean {
    const compareObj = configObjects[0];
    return configObjects.every(
      (cfg: ConfigObject) => cfg.crawlScheduleConfig.validFrom === compareObj.crawlScheduleConfig.validFrom);
  }

  static commonValidTo(configObjects: ConfigObject[]): boolean {
    const compareObj = configObjects[0];
    return configObjects.every(
      (cfg: ConfigObject) => cfg.crawlScheduleConfig.validTo === compareObj.crawlScheduleConfig.validTo);
  }

  static toProto(crawlScheduleConfig: CrawlScheduleConfig): CrawlScheduleConfigProto {
    return create(CrawlScheduleConfigSchema, {
      cronExpression: crawlScheduleConfig.cronExpression,
      validFrom: toTimestampProto(crawlScheduleConfig.validFrom),
      validTo: toTimestampProto(crawlScheduleConfig.validTo),
    });
  }
}
