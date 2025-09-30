import {ComponentFixture, TestBed} from '@angular/core/testing';

import {SchedulePreviewComponent} from './schedule-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('SchedulePreviewComponent', () => {
  let component: SchedulePreviewComponent;
  let fixture: ComponentFixture<SchedulePreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [SchedulePreviewComponent],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(SchedulePreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.CRAWLSCHEDULECONFIG});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
