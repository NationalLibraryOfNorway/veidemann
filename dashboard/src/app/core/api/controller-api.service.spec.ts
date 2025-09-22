import {inject, TestBed} from '@angular/core/testing';
import {ControllerApiService} from './controller-api.service';
import {provideCoreTesting} from '../core.testing.module';

describe('ControllerApiService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ControllerApiService,
        ...provideCoreTesting
      ]
    });
  });

  it('should be created', inject([ControllerApiService], (service: ControllerApiService) => {
    expect(service).toBeTruthy();
  }));
});
