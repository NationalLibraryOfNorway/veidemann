import {KindService} from './kind.service';
import {LabelService} from './label.service';
import { provideZonelessChangeDetection } from '@angular/core';
import { TestBed } from '@angular/core/testing';

describe('KindService', () => {
    let service: KindService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideZonelessChangeDetection(),
        {provide: LabelService, useValue: {}}
      ],
    });

    service = TestBed.inject(KindService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
