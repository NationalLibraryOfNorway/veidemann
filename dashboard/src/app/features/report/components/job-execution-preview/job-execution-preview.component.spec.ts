import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobExecutionPreviewComponent} from './job-execution-preview.component';
import {JobExecutionStatus} from '../../../shared/models';
import {RouterTestingModule} from '@angular/router/testing';
import {ActivatedRoute} from '@angular/router';
import {CommonsModule} from '../../../commons';
import {NGX_ECHARTS_CONFIG, NgxEchartsModule} from 'ngx-echarts';
import {CoreTestingModule} from '../../../core/core.testing.module';

describe('JobExecutionPreviewComponent', () => {
  let component: JobExecutionPreviewComponent;
  let fixture: ComponentFixture<JobExecutionPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [JobExecutionPreviewComponent],
      imports: [CoreTestingModule.forRoot(), CommonsModule, NgxEchartsModule, RouterTestingModule],
      providers: [
        {provide: NGX_ECHARTS_CONFIG, useValue: {}},
        {provide: ActivatedRoute, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobExecutionPreviewComponent);
    component = fixture.componentInstance;
    component.jobExecutionStatus = new JobExecutionStatus();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
