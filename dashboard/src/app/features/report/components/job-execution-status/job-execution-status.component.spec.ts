import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobExecutionStatusComponent} from './job-execution-status.component';
import { provideZonelessChangeDetection } from '@angular/core';

describe('JobExecutionStatusComponent', () => {
  let component: JobExecutionStatusComponent;
  let fixture: ComponentFixture<JobExecutionStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [JobExecutionStatusComponent],
      providers: [
        provideZonelessChangeDetection()
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
