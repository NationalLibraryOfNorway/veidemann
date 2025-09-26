import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PageLogShortcutsComponent} from './page-log-shortcuts.component';

describe('PageLogShortcutsComponent', () => {
  let component: PageLogShortcutsComponent;
  let fixture: ComponentFixture<PageLogShortcutsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [PageLogShortcutsComponent],
      imports: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogShortcutsComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
