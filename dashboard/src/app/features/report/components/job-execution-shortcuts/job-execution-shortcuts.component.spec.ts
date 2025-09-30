import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobExecutionShortcutsComponent} from './job-execution-shortcuts.component';
import {JobExecutionStatus} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('JobExecutionShortcutsComponent', () => {
  let component: JobExecutionShortcutsComponent;
  let fixture: ComponentFixture<JobExecutionShortcutsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [JobExecutionShortcutsComponent],
      providers: [
        ...provideCoreTesting,
      ]
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
