import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionStatusComponent} from './crawl-execution-status.component';
import {CrawlExecutionStatus} from '../../../shared/models';
import {RouterTestingModule} from '@angular/router/testing';
import {MatExpansionPanel} from '@angular/material/expansion';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {CommonsModule} from '../../../commons';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CrawlExecutionStatusComponent', () => {
  let component: CrawlExecutionStatusComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CommonsModule, RouterTestingModule, NoopAnimationsModule],
      declarations: [CrawlExecutionStatusComponent],
      providers: [
        MatExpansionPanel,
        provideZonelessChangeDetection()
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusComponent);
    component = fixture.componentInstance;
    component.crawlExecutionStatus = new CrawlExecutionStatus();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

});
