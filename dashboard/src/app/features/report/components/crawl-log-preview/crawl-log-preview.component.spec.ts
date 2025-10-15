import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogPreviewComponent} from './crawl-log-preview.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlLogPreviewComponent', () => {
  let component: CrawlLogPreviewComponent;
  let fixture: ComponentFixture<CrawlLogPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlLogPreviewComponent],
      providers: [
        ...provideCoreTesting,
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
