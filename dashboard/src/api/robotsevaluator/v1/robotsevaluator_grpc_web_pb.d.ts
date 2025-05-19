import * as grpcWeb from 'grpc-web';

import * as robotsevaluator_v1_robotsevaluator_pb from '../../robotsevaluator/v1/robotsevaluator_pb'; // proto import: "robotsevaluator/v1/robotsevaluator.proto"


export class RobotsEvaluatorClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  isAllowed(
    request: robotsevaluator_v1_robotsevaluator_pb.IsAllowedRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: robotsevaluator_v1_robotsevaluator_pb.IsAllowedReply) => void
  ): grpcWeb.ClientReadableStream<robotsevaluator_v1_robotsevaluator_pb.IsAllowedReply>;

}

export class RobotsEvaluatorPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  isAllowed(
    request: robotsevaluator_v1_robotsevaluator_pb.IsAllowedRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<robotsevaluator_v1_robotsevaluator_pb.IsAllowedReply>;

}

