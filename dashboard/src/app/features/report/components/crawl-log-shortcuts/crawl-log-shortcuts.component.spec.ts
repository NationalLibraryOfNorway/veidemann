import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogShortcutsComponent} from './crawl-log-shortcuts.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlLogShortcutsComponent', () => {
  let component: CrawlLogShortcutsComponent;
  let fixture: ComponentFixture<CrawlLogShortcutsComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CrawlLogShortcutsComponent],
      providers: [
        ...provideCoreTesting
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogShortcutsComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
