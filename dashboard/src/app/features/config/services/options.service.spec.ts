import {TestBed} from '@angular/core/testing';
import {OptionsService} from './options.service';
import {provideCoreTesting} from '../../../core/core.testing.module';

describe('OptionsService', () => {

  let service: OptionsService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ...provideCoreTesting
      ],
    });

    service = TestBed.inject(OptionsService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
