import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ReportComponent} from './report.component';
import {provideCoreTesting} from '../../core/core.testing.module';
import {provideRouter} from '@angular/router';


describe('ReportComponent', () => {
  let component: ReportComponent;
  let fixture: ComponentFixture<ReportComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ReportComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
      ]
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

  it('does not render a persistent report drawer', () => {
    expect(fixture.nativeElement.querySelector('mat-drawer')).toBeNull();
  });
});
