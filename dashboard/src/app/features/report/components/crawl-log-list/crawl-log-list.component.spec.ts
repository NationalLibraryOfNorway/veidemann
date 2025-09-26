import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogListComponent} from './crawl-log-list.component';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CrawlLogListComponent', () => {
  let component: CrawlLogListComponent;
  let fixture: ComponentFixture<CrawlLogListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        KeyboardShortcutsModule,
        NoopAnimationsModule
      ],
      providers: [
        provideZonelessChangeDetection()
      ],
      declarations: [CrawlLogListComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
