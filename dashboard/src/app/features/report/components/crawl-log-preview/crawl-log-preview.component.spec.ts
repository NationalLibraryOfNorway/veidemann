import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogPreviewComponent} from './crawl-log-preview.component';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CrawlLogPreviewComponent', () => {
  let component: CrawlLogPreviewComponent;
  let fixture: ComponentFixture<CrawlLogPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [CrawlLogPreviewComponent],
      providers: [
        provideZonelessChangeDetection()
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogPreviewComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
