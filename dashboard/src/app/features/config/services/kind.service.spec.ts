import {KindService} from './kind.service';
import {TestBed} from '@angular/core/testing';
import {provideCoreTesting} from '../../../core/core.testing.module';

describe('KindService', () => {
  let service: KindService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ...provideCoreTesting,
      ],
    });

    service = TestBed.inject(KindService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
