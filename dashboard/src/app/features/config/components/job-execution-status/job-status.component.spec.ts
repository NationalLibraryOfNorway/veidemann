import {ComponentFixture, TestBed} from '@angular/core/testing';

import {JobStatusComponent} from './job-status.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('JobStatusComponent', () => {
  let component: JobStatusComponent;
  let fixture: ComponentFixture<JobStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [JobStatusComponent],
      providers: [
        ...provideCoreTesting
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
