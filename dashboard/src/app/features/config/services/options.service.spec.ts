import { TestBed } from '@angular/core/testing';
import {LabelService} from './label.service';
import {OptionsService} from './options.service';
import { provideZonelessChangeDetection } from '@angular/core';

describe('OptionsService', () => {

  let service: OptionsService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideZonelessChangeDetection(),
        OptionsService,
        LabelService,
      ],
    });

    service = TestBed.inject(OptionsService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
