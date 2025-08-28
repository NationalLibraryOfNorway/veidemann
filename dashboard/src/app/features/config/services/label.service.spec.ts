import {TestBed} from '@angular/core/testing';

import {LabelService} from './label.service';
import {ConfigApiService} from '../../core';
import { provideZonelessChangeDetection } from '@angular/core';

describe('LabelService', () => {
  beforeEach(() => TestBed.configureTestingModule({
    providers: [
      provideZonelessChangeDetection(),
      LabelService,
      {
        provide: ConfigApiService,
        useValue: {}
      }
    ]
  }));

  it('should be created', () => {
    const service: LabelService = TestBed.inject(LabelService);
    expect(service).toBeTruthy();
  });
});
