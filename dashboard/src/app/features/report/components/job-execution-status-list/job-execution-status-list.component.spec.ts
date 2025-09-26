import {ComponentFixture, TestBed} from '@angular/core/testing';
import { provideZonelessChangeDetection } from '@angular/core';

import {JobExecutionStatusListComponent} from './job-execution-status-list.component';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';

describe('JobExecutionStatusListComponent', () => {
  let component: JobExecutionStatusListComponent;
  let fixture: ComponentFixture<JobExecutionStatusListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [KeyboardShortcutsModule, NoopAnimationsModule],
      declarations: [JobExecutionStatusListComponent],
      providers: [
        provideZonelessChangeDetection(),
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobExecutionStatusListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
