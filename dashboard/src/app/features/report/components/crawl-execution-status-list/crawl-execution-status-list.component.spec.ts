import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionStatusListComponent} from './crawl-execution-status-list.component';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CrawlExecutionStatusListComponent', () => {
  let component: CrawlExecutionStatusListComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [KeyboardShortcutsModule, NoopAnimationsModule],
      providers: [
        provideZonelessChangeDetection()
      ],
      declarations: [CrawlExecutionStatusListComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
