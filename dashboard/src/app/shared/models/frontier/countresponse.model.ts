import {CountResponse as CountResponseProto} from '../../../../api/frontier/v1/frontier_pb';

export class CountResponse {
  count: number;

  constructor({
                count = 0
              }: Partial<CountResponse> = {}) {
    this.count = count;
  }

  static fromProto(proto: CountResponseProto) {
    const countResponse = new CountResponse({
      count: Number(proto.count)
    });
    return countResponse;
  }
}
