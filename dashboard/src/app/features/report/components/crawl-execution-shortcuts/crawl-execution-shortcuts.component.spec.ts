import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionShortcutsComponent} from './crawl-execution-shortcuts.component';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {CrawlExecutionStatus} from '../../../../shared/models';

describe('CrawlExecutionShortcutsComponent', () => {
  let component: CrawlExecutionShortcutsComponent;
  let fixture: ComponentFixture<CrawlExecutionShortcutsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [CrawlExecutionShortcutsComponent],
      imports: [NoopAnimationsModule]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionShortcutsComponent);
    component = fixture.componentInstance;
    component.crawlExecutionStatus = new CrawlExecutionStatus();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
