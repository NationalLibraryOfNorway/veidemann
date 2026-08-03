import {create} from '@bufbuild/protobuf';
import {
  CrawlJob as CrawlJobProto,
  CrawlJobSchema,
  CrawlLimitsConfig as CrawlLimitsConfigProto,
  CrawlLimitsConfigSchema,
} from '../../../../api/config/v1/resources_pb';
import {ConfigRef} from './configref.model';
import {ConfigObject} from './configobject.model';
import {Kind} from './kind.model';

export class CrawlLimitsConfig {
  maxDurationS?: number; // int64
  maxBytes?: number; // int64

  constructor({
                maxDurationS = 0,
                maxBytes = 0,
              }: Partial<CrawlLimitsConfig> = {}) {
    this.maxDurationS = maxDurationS;
    this.maxBytes = maxBytes;
  }

  static fromProto(proto: CrawlLimitsConfigProto): CrawlLimitsConfig {
    return new CrawlLimitsConfig({
      maxDurationS: Number(proto.maxDurationS),
      maxBytes: Number(proto.maxBytes)
    });
  }

  static toProto(crawlLimitsConfig): CrawlLimitsConfigProto {
    return create(CrawlLimitsConfigSchema, {
      maxDurationS: BigInt(crawlLimitsConfig.maxDurationS || 0),
      maxBytes: BigInt(crawlLimitsConfig.maxBytes || 0),
    });
  }
}

export class CrawlJob {
  scheduleRef?: ConfigRef;
  crawlConfigRef?: ConfigRef;
  scopeScriptRef?: ConfigRef;
  limits: CrawlLimitsConfig;
  disabled: boolean;

  constructor({
                scheduleRef,
                crawlConfigRef,
                scopeScriptRef,
                limits,
                disabled = false
              }: Partial<CrawlJob> = {}) {
    this.scheduleRef = new ConfigRef(scheduleRef || {kind: Kind.CRAWLSCHEDULECONFIG});
    this.crawlConfigRef = new ConfigRef(crawlConfigRef || {kind: Kind.CRAWLCONFIG});
    this.scopeScriptRef = new ConfigRef(scopeScriptRef || {kind: Kind.BROWSERSCRIPT});
    this.limits = new CrawlLimitsConfig(limits);
    this.disabled = disabled;
  }

  static fromProto(proto: CrawlJobProto): CrawlJob {
    return new CrawlJob({
      scheduleRef: proto.scheduleRef ? ConfigRef.fromProto(proto.scheduleRef) : undefined,
      crawlConfigRef: proto.crawlConfigRef ? ConfigRef.fromProto(proto.crawlConfigRef) : undefined,
      scopeScriptRef: proto.scopeScriptRef ? ConfigRef.fromProto(proto.scopeScriptRef) : undefined,
      limits: proto.limits ? CrawlLimitsConfig.fromProto(proto.limits) : undefined,
      disabled: proto.disabled,
    });
  }

  static toProto(crawlJob: CrawlJob): CrawlJobProto {
    return create(CrawlJobSchema, {
      scheduleRef: ConfigRef.toProto(crawlJob.scheduleRef),
      scopeScriptRef: ConfigRef.toProto(crawlJob.scopeScriptRef),
      crawlConfigRef: ConfigRef.toProto(crawlJob.crawlConfigRef),
      limits: CrawlLimitsConfig.toProto(crawlJob.limits),
      disabled: crawlJob.disabled,
    });
  }

  static mergeConfigs(configObjects: ConfigObject[]): CrawlJob {
    const crawlJob = new CrawlJob({});
    const compareObj: CrawlJob = configObjects[0].crawlJob;

    const equalDisabledStatus = configObjects.every((cfg: ConfigObject) => cfg.crawlJob.disabled === compareObj.disabled);

    const equalMaxDuration = configObjects.every(
      (cfg: ConfigObject) => cfg.crawlJob.limits.maxDurationS === compareObj.limits.maxDurationS);

    const equalMaxBytes = configObjects.every((cfg: ConfigObject) => cfg.crawlJob.limits.maxBytes === compareObj.limits.maxBytes);

    const equalSchedule = configObjects.every((cfg: ConfigObject) => cfg.crawlJob.scheduleRef.id === compareObj.scheduleRef.id);

    const equalScopeScript = configObjects.every((cfg: ConfigObject) => cfg.crawlJob.scopeScriptRef.id === compareObj.scopeScriptRef.id);

    const equalCrawlConfig = configObjects.every(
      cfg => cfg.crawlJob.crawlConfigRef.id === compareObj.crawlConfigRef.id);

    if (equalDisabledStatus) {
      crawlJob.disabled = compareObj.disabled;
    } else {
      crawlJob.disabled = undefined;
    }

    if (equalMaxDuration) {
      crawlJob.limits.maxDurationS = compareObj.limits.maxDurationS;
    }

    if (equalMaxBytes) {
      crawlJob.limits.maxBytes = compareObj.limits.maxBytes;
    } else {
      crawlJob.limits.maxBytes = NaN;
    }

    crawlJob.scheduleRef = equalSchedule ? compareObj.scheduleRef : crawlJob.scheduleRef = new ConfigRef({kind: Kind.CRAWLSCHEDULECONFIG});

    crawlJob.crawlConfigRef = equalCrawlConfig ? compareObj.crawlConfigRef : new ConfigRef({kind: Kind.CRAWLCONFIG});

    crawlJob.scopeScriptRef = equalScopeScript ? compareObj.scopeScriptRef : new ConfigRef({kind: Kind.BROWSERSCRIPT});

    return crawlJob;
  }
}
