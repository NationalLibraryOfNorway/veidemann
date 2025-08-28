import {ComponentFixture, TestBed} from '@angular/core/testing';
import { provideRouter, RouterModule } from '@angular/router';
import { provideZonelessChangeDetection } from '@angular/core';

import {CrawlLogStatusComponent} from './crawl-log-status.component';
import {CommonsModule} from '../../../commons';
import {CrawlLog} from '../../../shared/models';

describe('CrawlLogStatusComponent', () => {
  let component: CrawlLogStatusComponent;
  let fixture: ComponentFixture<CrawlLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        RouterModule,
        CommonsModule
      ],
      providers: [
        provideZonelessChangeDetection(),
        provideRouter([])
      ],
      declarations: [CrawlLogStatusComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogStatusComponent);
    component = fixture.componentInstance;
    component.crawlLog = new CrawlLog();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
