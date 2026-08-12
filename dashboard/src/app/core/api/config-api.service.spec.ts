import {beforeEach, describe, expect, it} from 'vitest';
import {TestBed} from '@angular/core/testing';
import {provideCoreTesting} from '../core.testing.module';
import {ConfigApiService} from './config-api.service';
import {ApplicationErrorHandler} from '../error.handler';
import {Code, ConnectError} from '@connectrpc/connect';
import {firstValueFrom} from 'rxjs';
import {ConfigRef, Kind} from '../../shared/models';

describe('ConfigApiService', () => {
  let service: ConfigApiService;
  const handleError = vi.fn();

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ...provideCoreTesting,
        ConfigApiService,
        {provide: ApplicationErrorHandler, useValue: {handleError}},
      ],
    });

    service = TestBed.inject(ConfigApiService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('suppresses an expected not-found error for an optional config lookup', async () => {
    const getConfigObject = vi.fn().mockRejectedValue(new ConnectError('Seed not found', Code.NotFound));
    (service as unknown as {client: {getConfigObject: typeof getConfigObject}}).client = {getConfigObject};

    const result = await firstValueFrom(service.get(
      new ConfigRef({kind: Kind.SEED, id: 'deleted-seed-id'}),
      {suppressNotFound: true},
    ));

    expect(result).toBeNull();
    expect(handleError).not.toHaveBeenCalled();
  });
});


// describe('ConfigApiService', () => {
//   beforeEach(() => {
//     TestBed.configureTestingModule({
//       providers: [
//         ...provideCoreTesting,
//         ConfigApiService]
//     });
//   });
//
//   it('should be created', inject([ConfigApiService], (service: ConfigApiService) => {
//     expect(service).toBeTruthy();
//   }));

  // it('returns a list of configObjects', () => {
  // });
  //
  // it('returns the count of configObjects', () => {
  // });
  //
  // it('can get a single configObject', () => {
  // });
  //
  // it('save a configObject correct', () => {
  // });
  //
  // it('updates a configObject correctly', () => {
  // });
  //
  // it('deletes a single configObject', () => {
  // });
  //
  // it('fetches a list of label keys for a kind of configObject', () => {
  // });
  //
  // it('get a list of script annotations for a configObject', () => {
  // });
// });
