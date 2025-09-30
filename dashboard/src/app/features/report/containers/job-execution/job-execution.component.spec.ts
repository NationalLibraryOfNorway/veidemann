import {ComponentFixture, TestBed} from '@angular/core/testing';
import {JobExecutionComponent} from './job-execution.component';
import {JobExecutionService} from '../../services';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute} from '@angular/router';
import {of} from 'rxjs';
import {ControllerApiService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('JobExecutionComponent', () => {
  let component: JobExecutionComponent;
  let fixture: ComponentFixture<JobExecutionComponent>;

  const fakeActivatedRoute = {
    queryParamMap: of({
      get: () => {
      },
      getAll: () => {
      }
    }),
    snapshot: {
      data: {
        options: {}
      }
    },
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [JobExecutionComponent],
      declarations: [],
      providers: [
        ...provideCoreTesting,
        {provide: ActivatedRoute, useValue: fakeActivatedRoute},
        {provide: MatDialog, useValue: {}},
        {provide: JobExecutionService, useValue: {}},
        {provide: ControllerApiService, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(JobExecutionComponent);
    component = fixture.componentInstance;
  });

  it('should create', async () => {
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });
});
