import {TestBed} from '@angular/core/testing';
import {LabelService} from './label.service';
import {provideCoreTesting} from '../../../core/core.testing.module';
import {ConfigApiService} from '../../../core';
import {firstValueFrom, of} from 'rxjs';
import {Kind} from '../../../shared/models';

describe('LabelService', () => {
  const getLabelKeys = vi.fn(() => of(['owner']));

  beforeEach(() => TestBed.configureTestingModule({
    providers: [
      ...provideCoreTesting,
      {provide: ConfigApiService, useValue: {getLabelKeys}},
    ]
  }).compileComponents());

  beforeEach(() => getLabelKeys.mockClear());

  it('should be created', () => {
    const service: LabelService = TestBed.inject(LabelService);
    expect(service).toBeTruthy();
  });

  it.each([Kind.CRAWLENTITY, Kind.SEED, Kind.COLLECTION])(
    'requests label keys for kind %s',
    async kind => {
      const service = TestBed.inject(LabelService);

      await expect(firstValueFrom(service.getLabelKeys(kind))).resolves.toEqual(['owner']);
      expect(getLabelKeys).toHaveBeenCalledWith(expect.objectContaining({kind}));
    }
  );

  it('returns no keys without calling the API for an undefined kind', async () => {
    const service = TestBed.inject(LabelService);

    await expect(firstValueFrom(service.getLabelKeys(Kind.UNDEFINED))).resolves.toEqual([]);
    expect(getLabelKeys).not.toHaveBeenCalled();
  });
});
