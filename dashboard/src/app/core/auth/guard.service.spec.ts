import {inject, TestBed} from '@angular/core/testing';
import {GuardService} from './guard.service';
import {provideCoreTesting} from '../core.testing.module';

describe('GuardService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ...provideCoreTesting,
        GuardService
      ]
    });
  });

  it('should be created', inject([GuardService], (service: GuardService) => {
    expect(service).toBeTruthy();
  }));
});
