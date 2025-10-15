import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionPreviewComponent} from './crawl-execution-preview.component';
import {CrawlExecutionStatus} from '../../../../shared/models';
import {NGX_ECHARTS_CONFIG} from 'ngx-echarts';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlExecutionPreviewComponent', () => {
  let component: CrawlExecutionPreviewComponent;
  let fixture: ComponentFixture<CrawlExecutionPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlExecutionPreviewComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: NGX_ECHARTS_CONFIG,
          useValue: {}
        }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionPreviewComponent);
    component = fixture.componentInstance;
    component.crawlExecutionStatus = new CrawlExecutionStatus();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
