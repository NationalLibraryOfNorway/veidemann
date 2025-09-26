import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionPreviewComponent} from './crawl-execution-preview.component';
import {CrawlExecutionStatus} from '../../../../shared/models';
import {NgxFilesizeModule} from 'ngx-filesize';
import {NGX_ECHARTS_CONFIG, NgxEchartsModule} from 'ngx-echarts';

describe('CrawlExecutionPreviewComponent', () => {
  let component: CrawlExecutionPreviewComponent;
  let fixture: ComponentFixture<CrawlExecutionPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [CrawlExecutionPreviewComponent],
      imports: [NgxFilesizeModule, NgxEchartsModule],
      providers: [
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
