import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ReportComponent} from './report.component';
import {provideCoreTesting} from '../../core/core.testing.module';


describe('ReportComponent', () => {
  let component: ReportComponent;
  let fixture: ComponentFixture<ReportComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ReportComponent],
      providers: [
        ...provideCoreTesting,
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
});
