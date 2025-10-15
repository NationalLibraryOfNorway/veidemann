import * as grpcWeb from 'grpc-web';

import * as uricanonicalizer_v1_uricanonicalizer_pb from '../../uricanonicalizer/v1/uricanonicalizer_pb'; // proto import: "uricanonicalizer/v1/uricanonicalizer.proto"


export class UriCanonicalizerServiceClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  canonicalize(
    request: uricanonicalizer_v1_uricanonicalizer_pb.CanonicalizeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: uricanonicalizer_v1_uricanonicalizer_pb.CanonicalizeResponse) => void
  ): grpcWeb.ClientReadableStream<uricanonicalizer_v1_uricanonicalizer_pb.CanonicalizeResponse>;

}

export class UriCanonicalizerServicePromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  canonicalize(
    request: uricanonicalizer_v1_uricanonicalizer_pb.CanonicalizeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<uricanonicalizer_v1_uricanonicalizer_pb.CanonicalizeResponse>;

}

