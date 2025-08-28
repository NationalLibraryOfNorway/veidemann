import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ReportComponent} from './report.component';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {RouterTestingModule} from '@angular/router/testing';
import {ReportNavigationListComponent} from './containers/report-navigation-list/report-navigation-list.component';
import { CoreTestingModule } from '../../core/core.testing.module';


describe('ReportComponent', () => {
  let component: ReportComponent;
  let fixture: ComponentFixture<ReportComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [ReportComponent, ReportNavigationListComponent],
      imports: [CoreTestingModule.forRoot(), NoopAnimationsModule, RouterTestingModule]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ReportComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
