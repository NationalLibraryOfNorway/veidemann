import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobExecutionStatusComponent} from './job-execution-status.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('JobExecutionStatusComponent', () => {
  let component: JobExecutionStatusComponent;
  let fixture: ComponentFixture<JobExecutionStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [JobExecutionStatusComponent],
      providers: [
        ...provideCoreTesting,
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobExecutionStatusComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
