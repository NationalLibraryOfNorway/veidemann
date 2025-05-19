import * as grpcWeb from 'grpc-web';

import * as scopechecker_v1_scopechecker_pb from '../../scopechecker/v1/scopechecker_pb'; // proto import: "scopechecker/v1/scopechecker.proto"


export class ScopesCheckerServiceClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  scopeCheck(
    request: scopechecker_v1_scopechecker_pb.ScopeCheckRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: scopechecker_v1_scopechecker_pb.ScopeCheckResponse) => void
  ): grpcWeb.ClientReadableStream<scopechecker_v1_scopechecker_pb.ScopeCheckResponse>;

}

export class ScopesCheckerServicePromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  scopeCheck(
    request: scopechecker_v1_scopechecker_pb.ScopeCheckRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<scopechecker_v1_scopechecker_pb.ScopeCheckResponse>;

}

