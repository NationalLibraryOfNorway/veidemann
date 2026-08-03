import {create} from '@bufbuild/protobuf';
import {
  CrawlHostGroupConfig as CrawlHostGroupConfigProto,
  CrawlHostGroupConfigSchema,
} from '../../../../api/config/v1/resources_pb';
import {ConfigObject} from './configobject.model';
import {IpRange} from './ip-range.model';


export class CrawlHostGroupConfig {
  ipRangeList: IpRange[];
  minTimeBetweenPageLoadMs: number;
  maxTimeBetweenPageLoadMs: number;
  delayFactor: number;
  maxRetries: number;
  retryDelaySeconds: number;

  constructor({
                ipRangeList = [],
                minTimeBetweenPageLoadMs = 0,
                maxTimeBetweenPageLoadMs = 0,
                delayFactor = 0,
                maxRetries = 0,
                retryDelaySeconds = 0
              }: Partial<CrawlHostGroupConfig> = {}) {
    this.ipRangeList = ipRangeList ? ipRangeList.map(ipRange => new IpRange(ipRange)) : [];
    this.minTimeBetweenPageLoadMs = minTimeBetweenPageLoadMs;
    this.maxTimeBetweenPageLoadMs = maxTimeBetweenPageLoadMs;
    this.delayFactor = delayFactor;
    this.maxRetries = maxRetries;
    this.retryDelaySeconds = retryDelaySeconds;
  }

  static fromProto(proto: CrawlHostGroupConfigProto): CrawlHostGroupConfig {
    // a small hack, see https://github.com/grpc/grpc/issues/2227
    const delayFactor = parseFloat(proto.delayFactor.toPrecision(5));
    return new CrawlHostGroupConfig({
      ipRangeList: proto.ipRange.map(IpRange.fromProto),
      minTimeBetweenPageLoadMs: Number(proto.minTimeBetweenPageLoadMs),
      maxTimeBetweenPageLoadMs: Number(proto.maxTimeBetweenPageLoadMs),
      delayFactor,
      maxRetries: proto.maxRetries,
      retryDelaySeconds: proto.retryDelaySeconds
    });
  }

  static mergeConfigs(configObjects: ConfigObject[]): CrawlHostGroupConfig {
    const crawlHostGroupConfig = new CrawlHostGroupConfig();
    const compareObj: CrawlHostGroupConfig = configObjects[0].crawlHostGroupConfig;

    crawlHostGroupConfig.ipRangeList = configObjects.reduce((acc: any[], curr: ConfigObject) => {
      if (acc.length < 1) {
        acc = [...curr.crawlHostGroupConfig.ipRangeList];
      } else {
        acc = IpRange.intersectIpRange(acc, curr.crawlHostGroupConfig.ipRangeList || []);
      }
      return acc;
    }, []);

    const equalMinTimeBetweenPageload = configObjects.every(
      (cfg: ConfigObject) => cfg.crawlHostGroupConfig.minTimeBetweenPageLoadMs === compareObj.minTimeBetweenPageLoadMs);

    const equalMaxTimeBetweenPageload = configObjects.every(
      (cfg: ConfigObject) => cfg.crawlHostGroupConfig.maxTimeBetweenPageLoadMs === compareObj.maxTimeBetweenPageLoadMs);

    const equalDelayFactor = configObjects.every(
      (cfg: ConfigObject) => cfg.crawlHostGroupConfig.delayFactor === compareObj.delayFactor);

    const equalMaxRetries = configObjects.every(
      (cfg: ConfigObject) => cfg.crawlHostGroupConfig.maxRetries === compareObj.maxRetries);

    const equalRetryDelay = configObjects.every(
      (cfg: ConfigObject) => cfg.crawlHostGroupConfig.retryDelaySeconds === compareObj.retryDelaySeconds);

    if (equalMinTimeBetweenPageload) {
      crawlHostGroupConfig.minTimeBetweenPageLoadMs = compareObj.minTimeBetweenPageLoadMs;
    } else {
      crawlHostGroupConfig.minTimeBetweenPageLoadMs = null;
    }
    if (equalMaxTimeBetweenPageload) {
      crawlHostGroupConfig.maxTimeBetweenPageLoadMs = compareObj.maxTimeBetweenPageLoadMs;
    } else {
      crawlHostGroupConfig.maxTimeBetweenPageLoadMs = null;
    }
    if (equalDelayFactor) {
      crawlHostGroupConfig.delayFactor = compareObj.delayFactor;
    } else {
      crawlHostGroupConfig.delayFactor = null;
    }
    if (equalMaxRetries) {
      crawlHostGroupConfig.maxRetries = compareObj.maxRetries;
    } else {
      crawlHostGroupConfig.maxRetries = null;
    }
    if (equalRetryDelay) {
      crawlHostGroupConfig.retryDelaySeconds = compareObj.retryDelaySeconds;
    } else {
      crawlHostGroupConfig.retryDelaySeconds = null;
    }

    return crawlHostGroupConfig;
  }

  static toProto(crawlHostGroupConfig: CrawlHostGroupConfig) {
    return create(CrawlHostGroupConfigSchema, {
      ipRange: crawlHostGroupConfig.ipRangeList.map(IpRange.toProto),
      minTimeBetweenPageLoadMs: BigInt(crawlHostGroupConfig.minTimeBetweenPageLoadMs || 0),
      maxTimeBetweenPageLoadMs: BigInt(crawlHostGroupConfig.maxTimeBetweenPageLoadMs || 0),
      delayFactor: crawlHostGroupConfig.delayFactor,
      maxRetries: crawlHostGroupConfig.maxRetries,
      retryDelaySeconds: crawlHostGroupConfig.retryDelaySeconds,
    });
  }
}

