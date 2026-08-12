import {TestBed} from '@angular/core/testing';
import {firstValueFrom, of, throwError} from 'rxjs';

import {ConfigObject, ConfigRef, Kind, Meta} from '../../../shared/models';
import {ConfigService} from '../../../shared/services';
import {ConfigRefNamePipe} from './config-ref-name.pipe';

describe('ConfigRefNamePipe', () => {
  const get = vi.fn();
  let pipe: ConfigRefNamePipe;

  beforeEach(() => {
    get.mockReset();
    TestBed.configureTestingModule({
      providers: [
        ConfigRefNamePipe,
        {provide: ConfigService, useValue: {get}},
      ],
    });
    pipe = TestBed.inject(ConfigRefNamePipe);
  });

  it('loads the referenced configuration name', async () => {
    const ref = new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'});
    get.mockReturnValue(of(new ConfigObject({
      id: ref.id,
      kind: ref.kind,
      meta: new Meta({name: 'News collection'}),
    })));

    await expect(firstValueFrom(pipe.transform(ref))).resolves.toBe('News collection');
    expect(get).toHaveBeenCalledWith(ref);
  });

  it('uses the ID only when the referenced configuration has no name', async () => {
    const ref = new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'});
    get.mockReturnValue(of(new ConfigObject({id: ref.id, kind: ref.kind})));

    await expect(firstValueFrom(pipe.transform(ref))).resolves.toBe(ref.id);
  });

  it('uses the ID when the referenced configuration cannot be loaded', async () => {
    const ref = new ConfigRef({kind: Kind.COLLECTION, id: 'missing'});
    get.mockReturnValue(throwError(() => new Error('missing')));

    await expect(firstValueFrom(pipe.transform(ref))).resolves.toBe(ref.id);
  });
});
