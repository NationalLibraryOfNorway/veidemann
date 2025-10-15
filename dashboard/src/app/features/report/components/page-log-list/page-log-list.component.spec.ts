import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PageLogListComponent} from './page-log-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';


describe('PageLogListComponent', () => {
  let component: PageLogListComponent;
  let fixture: ComponentFixture<PageLogListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        PageLogListComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
