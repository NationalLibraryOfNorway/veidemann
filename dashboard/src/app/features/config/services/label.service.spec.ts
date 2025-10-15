import {TestBed} from '@angular/core/testing';
import {LabelService} from './label.service';
import {provideCoreTesting} from '../../../core/core.testing.module';

describe('LabelService', () => {
  beforeEach(() => TestBed.configureTestingModule({
    providers: [
      ...provideCoreTesting,
    ]
  }));

  it('should be created', () => {
    const service: LabelService = TestBed.inject(LabelService);
    expect(service).toBeTruthy();
  });
});
