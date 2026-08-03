import {create} from '@bufbuild/protobuf';
import {
  ConfigObject as ConfigObjectProto,
  ConfigObjectSchema,
} from '../../../../api/config/v1/resources_pb';
import {intersectLabel} from '../../func/group-update/labels/common-labels';
import {Collection} from './collection.model';
import {CrawlEntity} from './crawlentity.model';
import {Seed} from './seed.model';
import {CrawlJob} from './crawljob.model';
import {CrawlConfig} from './crawlconfig.model';
import {CrawlScheduleConfig} from './crawlscheduleconfig.model';
import {BrowserConfig} from './browserconfig.model';
import {PolitenessConfig} from './politenessconfig.model';
import {BrowserScript} from './browserscript.model';
import {CrawlHostGroupConfig} from './crawlhostgroupconfig.model';
import {RoleMapping} from './rolemapping.model';
import {Kind} from './kind.model';
import {Meta} from './meta.model';
import {ConfigRef} from './configref.model';


export class ConfigObject {
  id: string;
  apiVersion: string;
  kind: Kind;
  meta: Meta;
  crawlEntity?: CrawlEntity;
  seed?: Seed;
  crawlJob?: CrawlJob;
  crawlConfig?: CrawlConfig;
  crawlScheduleConfig?: CrawlScheduleConfig;
  browserConfig?: BrowserConfig;
  politenessConfig?: PolitenessConfig;
  browserScript?: BrowserScript;
  crawlHostGroupConfig?: CrawlHostGroupConfig;
  roleMapping?: RoleMapping;
  collection?: Collection;

  constructor(configObject: Partial<ConfigObject> = {}) {
    this.id = configObject.id || '';
    this.apiVersion = configObject.apiVersion || 'v1';
    this.kind = configObject.kind || Kind.UNDEFINED;
    this.meta = new Meta(configObject.meta);

    switch (configObject.kind) {
      case Kind.UNDEFINED:
        break;
      case Kind.CRAWLENTITY:
        this.crawlEntity = new CrawlEntity();
        break;
      case Kind.SEED:
        this.seed = new Seed(configObject.seed);
        break;
      case Kind.CRAWLJOB:
        this.crawlJob = new CrawlJob(configObject.crawlJob);
        break;
      case Kind.CRAWLCONFIG:
        this.crawlConfig = new CrawlConfig(configObject.crawlConfig);
        break;
      case Kind.CRAWLSCHEDULECONFIG:
        this.crawlScheduleConfig = new CrawlScheduleConfig(configObject.crawlScheduleConfig);
        break;
      case Kind.BROWSERCONFIG:
        this.browserConfig = new BrowserConfig(configObject.browserConfig);
        break;
      case Kind.POLITENESSCONFIG:
        this.politenessConfig = new PolitenessConfig(configObject.politenessConfig);
        break;
      case Kind.BROWSERSCRIPT:
        this.browserScript = new BrowserScript(configObject.browserScript);
        break;
      case Kind.CRAWLHOSTGROUPCONFIG:
        this.crawlHostGroupConfig = new CrawlHostGroupConfig(configObject.crawlHostGroupConfig);
        break;
      case Kind.ROLEMAPPING:
        this.roleMapping = new RoleMapping(configObject.roleMapping);
        break;
      case Kind.COLLECTION:
        this.collection = new Collection(configObject.collection);
        break;
    }
  }

  static fromProto(proto: ConfigObjectProto): ConfigObject {
    const config = new ConfigObject({
      id: proto.id,
      apiVersion: proto.apiVersion,
      kind: proto.kind.valueOf(),
      meta: proto.meta ? Meta.fromProto(proto.meta) : undefined,
    });
    switch (proto.spec.case) {
      case 'crawlEntity': config.crawlEntity = CrawlEntity.fromProto(proto.spec.value); break;
      case 'seed': config.seed = Seed.fromProto(proto.spec.value); break;
      case 'crawlJob': config.crawlJob = CrawlJob.fromProto(proto.spec.value); break;
      case 'crawlConfig': config.crawlConfig = CrawlConfig.fromProto(proto.spec.value); break;
      case 'crawlScheduleConfig': config.crawlScheduleConfig = CrawlScheduleConfig.fromProto(proto.spec.value); break;
      case 'browserConfig': config.browserConfig = BrowserConfig.fromProto(proto.spec.value); break;
      case 'politenessConfig': config.politenessConfig = PolitenessConfig.fromProto(proto.spec.value); break;
      case 'browserScript': config.browserScript = BrowserScript.fromProto(proto.spec.value); break;
      case 'crawlHostGroupConfig': config.crawlHostGroupConfig = CrawlHostGroupConfig.fromProto(proto.spec.value); break;
      case 'roleMapping': config.roleMapping = RoleMapping.fromProto(proto.spec.value); break;
      case 'collection': config.collection = Collection.fromProto(proto.spec.value); break;
    }
    return config;
  }

  static toProto(configObject: ConfigObject): ConfigObjectProto {
    let spec: ConfigObjectProto['spec'] = {case: undefined};
    if (configObject.crawlEntity) {
      spec = {case: 'crawlEntity', value: CrawlEntity.toProto(configObject.crawlEntity)};
    } else if (configObject.seed) {
      spec = {case: 'seed', value: Seed.toProto(configObject.seed)};
    } else if (configObject.crawlJob) {
      spec = {case: 'crawlJob', value: CrawlJob.toProto(configObject.crawlJob)};
    } else if (configObject.crawlConfig) {
      spec = {case: 'crawlConfig', value: CrawlConfig.toProto(configObject.crawlConfig)};
    } else if (configObject.crawlScheduleConfig) {
      spec = {case: 'crawlScheduleConfig', value: CrawlScheduleConfig.toProto(configObject.crawlScheduleConfig)};
    } else if (configObject.browserConfig) {
      spec = {case: 'browserConfig', value: BrowserConfig.toProto(configObject.browserConfig)};
    } else if (configObject.politenessConfig) {
      spec = {case: 'politenessConfig', value: PolitenessConfig.toProto(configObject.politenessConfig)};
    } else if (configObject.browserScript) {
      spec = {case: 'browserScript', value: BrowserScript.toProto(configObject.browserScript)};
    } else if (configObject.crawlHostGroupConfig) {
      spec = {case: 'crawlHostGroupConfig', value: CrawlHostGroupConfig.toProto(configObject.crawlHostGroupConfig)};
    } else if (configObject.roleMapping) {
      spec = {case: 'roleMapping', value: RoleMapping.toProto(configObject.roleMapping)};
    } else if (configObject.collection) {
      spec = {case: 'collection', value: Collection.toProto(configObject.collection)};
    }
    return create(ConfigObjectSchema, {
      apiVersion: configObject.apiVersion || 'v1',
      id: configObject.id,
      meta: Meta.toProto(configObject.meta),
      kind: configObject.kind.valueOf(),
      spec,
    });
  }

  static toConfigRef(configObject: ConfigObject): ConfigRef {
    return new ConfigRef({id: configObject.id, kind: configObject.kind});
  }

  static clone(configObject: ConfigObject): ConfigObject {
    const clone = new ConfigObject(configObject);
    clone.id = '';
    Object.assign(clone.meta, {created: '', createdBy: '', lastModified: '', lastModifiedBy: '', name: ''});
    return clone;
  }

  static mergeConfigs(configs: ConfigObject[]): ConfigObject {
    if (configs.length < 1) {
      return null;
    } else if (configs.length === 1) {
      return configs[0];
    }
    const configObject = new ConfigObject({kind: configs[0].kind});

    configObject.meta = new Meta();
    configObject.meta.labelList = configs.map(config => config.meta.labelList).reduce(intersectLabel);

    switch (configObject.kind) {
      case Kind.CRAWLENTITY:
        configObject.crawlEntity = CrawlEntity.mergeConfigs(configs);
        return configObject;
      case Kind.SEED:
        configObject.seed = Seed.merge(configs.map(config => config.seed));
        return configObject;
      case Kind.CRAWLJOB:
        configObject.crawlJob = CrawlJob.mergeConfigs(configs);
        return configObject;
      case Kind.CRAWLCONFIG:
        configObject.crawlConfig = CrawlConfig.mergeConfigs(configs);
        return configObject;
      case Kind.CRAWLSCHEDULECONFIG:
        configObject.crawlScheduleConfig = CrawlScheduleConfig.mergeConfigs(configs);
        return configObject;
      case Kind.BROWSERCONFIG:
        configObject.browserConfig = BrowserConfig.mergeConfigs(configs);
        return configObject;
      case Kind.POLITENESSCONFIG:
        configObject.politenessConfig = PolitenessConfig.mergeConfigs(configs);
        return configObject;
      case Kind.BROWSERSCRIPT:
        configObject.browserScript = BrowserScript.mergeConfigs(configs);
        return configObject;
      case Kind.CRAWLHOSTGROUPCONFIG:
        configObject.crawlHostGroupConfig = CrawlHostGroupConfig.mergeConfigs(configs);
        return configObject;
      case Kind.ROLEMAPPING:
        configObject.roleMapping = RoleMapping.mergeConfigs(configs);
        return configObject;
      case Kind.COLLECTION:
      case Kind.UNDEFINED:
      default:
        return null;
    }
  }
}
