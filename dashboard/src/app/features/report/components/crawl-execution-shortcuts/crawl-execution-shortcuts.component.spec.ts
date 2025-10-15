import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionShortcutsComponent} from './crawl-execution-shortcuts.component';
import {CrawlExecutionStatus} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlExecutionShortcutsComponent', () => {
  let component: CrawlExecutionShortcutsComponent;
  let fixture: ComponentFixture<CrawlExecutionShortcutsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionShortcutsComponent],
      providers: [
        ...provideCoreTesting,
      ]
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
