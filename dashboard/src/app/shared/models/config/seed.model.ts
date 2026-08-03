import {create} from '@bufbuild/protobuf';
import {Seed as SeedProto, SeedSchema} from '../../../../api/config/v1/resources_pb';
import {ConfigRef} from './configref.model';
import {Kind} from './kind.model';

export class Seed {
  entityRef: ConfigRef;
  jobRefList: ConfigRef[];
  disabled: boolean;

  constructor({
                entityRef,
                jobRefList = [],
                disabled = false
              }: Partial<Seed> = {}) {
    this.entityRef = entityRef || new ConfigRef({kind: Kind.CRAWLENTITY});
    this.jobRefList = jobRefList ? jobRefList.map(configRef => new ConfigRef(configRef)) : [];
    this.disabled = disabled;
  }

  static fromProto(proto: SeedProto): Seed {
    return new Seed({
      entityRef: proto.entityRef ? ConfigRef.fromProto(proto.entityRef) : new ConfigRef({kind: Kind.CRAWLENTITY}),
      jobRefList: proto.jobRef.map(ref => ConfigRef.fromProto(ref)),
      disabled: proto.disabled
    });
  }

  static toProto(seed: Seed): SeedProto {
    const entityRef = seed.entityRef?.id
      ? ConfigRef.toProto(new ConfigRef({kind: Kind.CRAWLENTITY, id: seed.entityRef.id}))
      : undefined;
    return create(SeedSchema, {
      entityRef,
      jobRef: seed.jobRefList.map(ConfigRef.toProto),
      disabled: seed.disabled,
    });
  }

  static merge(seeds: Seed[]): Seed {
    const mergedSeed = new Seed();
    const compareObj: Seed = seeds[0];
    const commonCrawljobs = this.commonCrawlJobRefs(seeds);

    mergedSeed.disabled = seeds.every(seed => seed.disabled === compareObj.disabled)
      ? compareObj.disabled
      : undefined;

    for (const crawlJob of commonCrawljobs) {
      const gotJob = seeds.every((cfg) =>
        cfg.jobRefList.some(jobRef => jobRef.id === crawlJob.id));
      if (gotJob) {
        mergedSeed.jobRefList.push(crawlJob);
      }
    }

    return mergedSeed;
  }

  static commonCrawlJobRefs(seeds: Seed[]): ConfigRef[] {
    return seeds
      .map(seed => seed.jobRefList)
      .reduce((acc, curr) => acc.concat(curr), [])
      .filter(function addIfNotPresent({id}, _, arr) {
        return !this.has(id) && this.add(id);
      }, new Set());
  }
}
