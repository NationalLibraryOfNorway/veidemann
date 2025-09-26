import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobExecutionShortcutsComponent} from './job-execution-shortcuts.component';
import {JobExecutionStatus} from '../../../../shared/models';

describe('JobExecutionShortcutsComponent', () => {
  let component: JobExecutionShortcutsComponent;
  let fixture: ComponentFixture<JobExecutionShortcutsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [JobExecutionShortcutsComponent],
      imports: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobExecutionShortcutsComponent);
    component = fixture.componentInstance;
    component.jobExecutionStatus = new JobExecutionStatus();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
