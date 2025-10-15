import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogStatusComponent} from './crawl-log-status.component';
import {CrawlLog} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';

describe('CrawlLogStatusComponent', () => {
  let component: CrawlLogStatusComponent;
  let fixture: ComponentFixture<CrawlLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CrawlLogStatusComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {provide: ActivatedRoute, useValue: {}}
      ],

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
