import {inject, TestBed} from '@angular/core/testing';
import { HttpClient } from '@angular/common/http';
import {ReportApiService} from './report-api.service';
import {provideCoreTesting} from '../core.testing.module';

describe('ReportApiService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ...provideCoreTesting,
        ReportApiService,
        {provide: HttpClient, useValue: {}}
      ]
    });
  });

  it('should be created', inject([ReportApiService], (service: ReportApiService) => {
    expect(service).toBeTruthy();
  }));
});
