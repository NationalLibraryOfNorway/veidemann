import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobStatusComponent} from './job-status.component';
import { provideZonelessChangeDetection } from '@angular/core';

describe('JobStatusComponent', () => {
  let component: JobStatusComponent;
  let fixture: ComponentFixture<JobStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [JobStatusComponent],
      providers: [
        provideZonelessChangeDetection()
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(JobStatusComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
