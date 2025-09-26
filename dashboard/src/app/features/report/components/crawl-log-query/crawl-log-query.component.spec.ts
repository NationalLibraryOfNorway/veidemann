import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogQueryComponent} from './crawl-log-query.component';
import {UntypedFormBuilder, ReactiveFormsModule} from '@angular/forms';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CrawlLogQueryComponent', () => {
  let component: CrawlLogQueryComponent;
  let fixture: ComponentFixture<CrawlLogQueryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, ReactiveFormsModule],
      declarations: [CrawlLogQueryComponent],
      providers: [
        provideZonelessChangeDetection(),
        UntypedFormBuilder]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogQueryComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
