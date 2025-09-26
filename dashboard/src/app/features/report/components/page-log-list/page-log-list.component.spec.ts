import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PageLogListComponent} from './page-log-list.component';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';


describe('PageLogListComponent', () => {
  let component: PageLogListComponent;
  let fixture: ComponentFixture<PageLogListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        KeyboardShortcutsModule,
        NoopAnimationsModule
      ],
      declarations: [PageLogListComponent],
      providers: [
        provideZonelessChangeDetection()
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
