import {create} from '@bufbuild/protobuf';
import {
  CrawlConfig as CrawlConfigProto,
  CrawlConfigSchema,
  ExtraConfig as ExtraConfigProto,
  ExtraConfigSchema,
} from '../../../../api/config/v1/resources_pb';
import {ConfigRef} from './configref.model';
import {Kind} from './kind.model';
import {ConfigObject} from './configobject.model';

export class ExtraConfig {
  createScreenshot: boolean;

  constructor({createScreenshot = true}: Partial<ExtraConfig> = {}) {
    this.createScreenshot = createScreenshot;
  }

  static fromProto(proto: ExtraConfigProto): ExtraConfig {
    return new ExtraConfig({
      createScreenshot: proto.createScreenshot
    });
  }

  static toProto(extraConfig: ExtraConfig): ExtraConfigProto {
    return create(ExtraConfigSchema, {createScreenshot: extraConfig.createScreenshot});
  }
}

export class CrawlConfig {
  collectionRef: ConfigRef;
  browserConfigRef: ConfigRef;
  politenessRef: ConfigRef;
  extra: ExtraConfig;
  minimumDnsTtlS: number;
  priorityWeight: number;

  constructor({
                extra,
                browserConfigRef,
                politenessRef,
                collectionRef,
                minimumDnsTtlS = 0,
                priorityWeight = 0
              }: Partial<CrawlConfig> = {}) {
    this.browserConfigRef = new ConfigRef(browserConfigRef || {kind: Kind.BROWSERCONFIG});
    this.politenessRef = new ConfigRef(politenessRef || {kind: Kind.POLITENESSCONFIG});
    this.collectionRef = new ConfigRef(collectionRef || {kind: Kind.COLLECTION});
    this.extra = new ExtraConfig(extra);
    this.minimumDnsTtlS = minimumDnsTtlS;
    this.priorityWeight = priorityWeight;
  }

  static fromProto(proto: CrawlConfigProto): CrawlConfig {
    return new CrawlConfig({
      collectionRef: proto.collectionRef ? ConfigRef.fromProto(proto.collectionRef) : null,
      browserConfigRef: proto.browserConfigRef ? ConfigRef.fromProto(proto.browserConfigRef) : null,
      politenessRef: proto.politenessRef ? ConfigRef.fromProto(proto.politenessRef) : null,
      extra: proto.extra ? ExtraConfig.fromProto(proto.extra) : undefined,
      minimumDnsTtlS: proto.minimumDnsTtlS,
      priorityWeight: proto.priorityWeight,
    });
  }

  static toProto(crawlConfig: CrawlConfig): CrawlConfigProto {
    return create(CrawlConfigSchema, {
      collectionRef: ConfigRef.toProto(crawlConfig.collectionRef),
      browserConfigRef: ConfigRef.toProto(crawlConfig.browserConfigRef),
      politenessRef: ConfigRef.toProto(crawlConfig.politenessRef),
      extra: ExtraConfig.toProto(crawlConfig.extra),
      minimumDnsTtlS: crawlConfig.minimumDnsTtlS,
      priorityWeight: crawlConfig.priorityWeight,
    });
  }

  static mergeConfigs(configObjects: ConfigObject[]): CrawlConfig {
    const crawlConfig = new CrawlConfig();
    const compareObj: CrawlConfig = configObjects[0].crawlConfig;

    const equalCollectionRef = configObjects.every(
      cfg => cfg.crawlConfig.collectionRef.id === compareObj.collectionRef.id);

    const equalBrowserConfigRef = configObjects.every(
      cfg => cfg.crawlConfig.browserConfigRef.id === compareObj.browserConfigRef.id);

    const equalPolitenessRef = configObjects.every(cfg => cfg.crawlConfig.politenessRef.id === compareObj.politenessRef.id);

    const equalDnsTtlS = configObjects.every(cfg => cfg.crawlConfig.minimumDnsTtlS === compareObj.minimumDnsTtlS);

    const equalPriorityWeight = configObjects.every(cfg => cfg.crawlConfig.priorityWeight === compareObj.priorityWeight);

    const equalCreateScreenshot = configObjects.every(
      cfg => cfg.crawlConfig.extra.createScreenshot === compareObj.extra.createScreenshot);

    if (equalCollectionRef) {
      crawlConfig.collectionRef = compareObj.collectionRef;
    } else {
      crawlConfig.collectionRef = null;
    }

    if (equalBrowserConfigRef) {
      crawlConfig.browserConfigRef = compareObj.browserConfigRef;
    } else {
      crawlConfig.browserConfigRef = null;
    }

    if (equalPolitenessRef) {
      crawlConfig.politenessRef = compareObj.politenessRef;
    } else {
      crawlConfig.politenessRef = null;
    }

    if (equalDnsTtlS) {
      crawlConfig.minimumDnsTtlS = compareObj.minimumDnsTtlS;
    }

    if (equalPriorityWeight) {
      crawlConfig.priorityWeight = compareObj.priorityWeight;
    }

    if (equalCreateScreenshot) {
      crawlConfig.extra.createScreenshot = compareObj.extra.createScreenshot;
    } else {
      crawlConfig.extra.createScreenshot = null;
    }

    return crawlConfig;
  }
}
