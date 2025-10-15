import * as grpcWeb from 'grpc-web';

import * as dnsresolver_v1_dnsresolver_pb from '../../dnsresolver/v1/dnsresolver_pb'; // proto import: "dnsresolver/v1/dnsresolver.proto"


export class DnsResolverClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  resolve(
    request: dnsresolver_v1_dnsresolver_pb.ResolveRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: dnsresolver_v1_dnsresolver_pb.ResolveReply) => void
  ): grpcWeb.ClientReadableStream<dnsresolver_v1_dnsresolver_pb.ResolveReply>;

}

export class DnsResolverPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  resolve(
    request: dnsresolver_v1_dnsresolver_pb.ResolveRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<dnsresolver_v1_dnsresolver_pb.ResolveReply>;

}

