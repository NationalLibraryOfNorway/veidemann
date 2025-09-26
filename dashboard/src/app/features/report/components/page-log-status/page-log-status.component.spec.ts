import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PageLogStatusComponent} from './page-log-status.component';

describe('PageLogStatusComponent', () => {
  let component: PageLogStatusComponent;
  let fixture: ComponentFixture<PageLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [],
      declarations: [PageLogStatusComponent],
      providers: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogStatusComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
