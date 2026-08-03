import {create} from '@bufbuild/protobuf';
import {
  CrawlHostGroupConfig_IpRange as IpRangeProto,
  CrawlHostGroupConfig_IpRangeSchema,
} from '../../../../api/config/v1/resources_pb';

export class IpRange {
  ipFrom: string;
  ipTo: string;

  constructor({
                ipFrom = '',
                ipTo = ''
              }: Partial<IpRange> = {}) {
    this.ipFrom = ipFrom;
    this.ipTo = ipTo;
  }

  static fromProto(proto: IpRangeProto): IpRange {
    return new IpRange({
      ipFrom: proto.ipFrom,
      ipTo: proto.ipTo
    });
  }

  static intersectIpRange(a: IpRange[], b: IpRange[]): IpRange[] {
    const setA = Array.from(new Set(a));
    const setB = Array.from(new Set(b));
    const intersection = new Set(setA.filter((x: IpRange) =>
      setB.find((range: IpRange) => x.ipFrom === range.ipFrom && x.ipTo === range.ipTo) !== undefined
    ));
    return Array.from(intersection) as IpRange[];
  }

  static toProto(ipRange: IpRange): IpRangeProto {
    return create(CrawlHostGroupConfig_IpRangeSchema, {ipFrom: ipRange.ipFrom, ipTo: ipRange.ipTo});
  }
}
